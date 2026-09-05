package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	usersvc "github.com/cloudreve/Cloudreve/v4/service/user"
	"github.com/gin-gonic/gin"
)

// Provider is the identifier of a third-party SSO provider.
type Provider string

const (
	// ProviderLogto uses the Logto OIDC provider (logto_config).
	ProviderLogto Provider = "logto"
	// ProviderOIDC uses a generic OIDC provider (oidc_config).
	ProviderOIDC Provider = "oidc"
	// ProviderQQ uses the QQ OAuth2 provider (qq_login_config).
	ProviderQQ Provider = "qq"
)

// ssoConfig is the shared shape of provider configuration stored in settings.
// All three providers are persisted as a JSON object with these keys so that
// the admin UI and the backend share a consistent layout.
type ssoConfig struct {
	Endpoint      string `json:"endpoint,omitempty"`
	AppID         string `json:"app_id,omitempty"`
	AppSecret     string `json:"app_secret,omitempty"`
	DirectSignIn  bool   `json:"direct_sign_in,omitempty"`
	DisplayName   string `json:"display_name,omitempty"`
	Scope         string `json:"scope,omitempty"`
	AuthURL       string `json:"auth_url,omitempty"`
	TokenURL      string `json:"token_url,omitempty"`
	UserInfoURL   string `json:"userinfo_url,omitempty"`
	RedirectURL   string `json:"redirect_url,omitempty"` // QQ only; optional override
}

// ScopePrepared returns the default scope list when none is configured.
func (c *ssoConfig) ScopePrepared() string {
	if strings.TrimSpace(c.Scope) != "" {
		return c.Scope
	}
	return "openid email profile"
}

// providerSetting resolves the per-provider enable flag and config keys.
type providerSetting struct {
	EnabledKey string
	ConfigKey  string
}

var providerSettings = map[Provider]providerSetting{
	ProviderLogto: {EnabledKey: "logto_enabled", ConfigKey: "logto_config"},
	ProviderOIDC:  {EnabledKey: "oidc_enabled", ConfigKey: "oidc_config"},
	ProviderQQ:    {EnabledKey: "qq_login", ConfigKey: "qq_login_config"},
}

// IsSupportedProvider reports whether the provider identifier is one of the
// supported SSO providers.
func IsSupportedProvider(p Provider) bool {
	_, ok := providerSettings[p]
	return ok
}

// UnsupportedProviderError builds the error returned for unknown providers.
func UnsupportedProviderError() error {
	return serializer.NewError(serializer.CodeParamErr, "Unsupported SSO provider", nil)
}

// ssoSettingKeys are the PRO setting keys that may not exist on installations
// upgraded before they were introduced. They are seeded lazily so that admin
// settings save (which updates existing rows) works out of the box.
var ssoSettingKeys = []string{"oidc_enabled", "oidc_config"}

// ssoStateSessionKey is the session key under which the OIDC state is stored.
const ssoStateSessionKey = "sso_state"

// ssoState is the value persisted in the user session to bind the authorize
// request to the callback and to protect against CSRF / state confusion.
type ssoState struct {
	Provider  Provider `json:"provider"`
	State     string   `json:"state"`
	Verifier  string   `json:"verifier,omitempty"`  // PKCE code_verifier, when used
	Challenge string   `json:"challenge,omitempty"` // PKCE code_challenge (S256)
}

// StartResponse is returned by the SSO start endpoint.
type StartResponse struct {
	Provider Provider `json:"provider"`
	URL      string   `json:"url"`
	State    string   `json:"state"`
}

// SSOService handles initiating and completing third-party SSO logins.
type SSOService struct {
	provider Provider
}

func NewSSOService(provider Provider) *SSOService {
	return &SSOService{provider: provider}
}

// SSOEnabled reports whether the given SSO provider is enabled in settings.
func SSOEnabled(c *gin.Context, dep dependency.Dep, provider Provider) bool {
	p, ok := providerSettings[provider]
	if !ok {
		return false
	}
	seedSSOSettings(c, dep)
	return ssoEnabledKey(c, dep, p.EnabledKey)
}

// Start initiates an SSO login for the provider and returns the authorization
// URL the browser should be redirected to.
func (s *SSOService) Start(c *gin.Context) (*StartResponse, error) {
	dep := dependency.FromContext(c)
	seedSSOSettings(c, dep)
	if !SSOEnabled(c, dep, s.provider) {
		return nil, serializer.NewError(serializer.CodeFeatureNotEnabled, "SSO provider is not enabled", nil)
	}

	cfg, err := s.config(c, dep)
	if err != nil {
		return nil, err
	}

	state, verifier, challenge, err := s.buildAuthorizeState(c)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to prepare SSO state", err)
	}

	authURL, err := s.authorizeURL(c, dep, cfg, state, challenge)
	if err != nil {
		return nil, err
	}

	_ = verifier
	return &StartResponse{
		Provider: s.provider,
		URL:      authURL,
		State:    state,
	}, nil
}

// Callback completes an SSO login after the provider redirects the browser
// back. It exchanges the authorization code, fetches the user profile, finds
// or auto-provisions the Cloudreve account, and issues the token pair.
func (s *SSOService) Callback(c *gin.Context, code, state string) (*usersvc.BuiltinLoginResponse, error) {
	dep := dependency.FromContext(c)
	seedSSOSettings(c, dep)
	if code == "" {
		return nil, serializer.NewError(serializer.CodeParamErr, "Missing authorization code", nil)
	}

	// Load and validate the state bound to the current session.
	expected, err := s.consumeState(c, state)
	if err != nil {
		return nil, err
	}

	cfg, err := s.config(c, dep)
	if err != nil {
		return nil, err
	}

	token, err := s.exchangeCode(c, dep, cfg, code, expected.Verifier, state)
	if err != nil {
		return nil, err
	}

	identifier, nickname, err := s.fetchUserInfo(c, dep, cfg, token)
	if err != nil {
		return nil, err
	}
	if identifier == "" {
		return nil, serializer.NewError(serializer.CodeCredentialInvalid, "Failed to resolve a unique user from the provider profile", nil)
	}

	u, err := s.findOrCreateUser(c, dep, identifier, nickname)
	if err != nil {
		return nil, err
	}

	tokenPair, err := dep.TokenAuth().Issue(c, &auth.IssueTokenArgs{User: u, RootTokenID: nil})
	if err != nil {
		return nil, serializer.NewError(serializer.CodeEncryptError, "Failed to issue token pair", err)
	}

	return &usersvc.BuiltinLoginResponse{
		User:  usersvc.BuildUser(u, dep.HashIDEncoder()),
		Token: *tokenPair,
	}, nil
}

// config reads and parses the provider's configuration from settings.
func (s *SSOService) config(c *gin.Context, dep dependency.Dep) (*ssoConfig, error) {
	p := providerSettings[s.provider]
	raw := getSettingString(c, dep, p.ConfigKey, "")
	cfg := &ssoConfig{}
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), cfg); err != nil {
			return nil, serializer.NewError(serializer.CodeInternalSetting, "Invalid SSO configuration", err)
		}
	}
	return cfg, nil
}

// buildAuthorizeState generates an opaque state and an optional PKCE pair, and
// stores them in the user session.
func (s *SSOService) buildAuthorizeState(c *gin.Context) (string, string, string, error) {
	state := util.RandStringRunesCrypto(32)
	verifier := util.RandStringRunesCrypto(48)

	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	record, err := json.Marshal(&ssoState{
		Provider:  s.provider,
		State:     state,
		Verifier:  verifier,
		Challenge: challenge,
	})
	if err != nil {
		return "", "", "", err
	}

	util.SetSession(c, map[string]interface{}{ssoStateSessionKey: string(record)})
	return state, verifier, challenge, nil
}

// consumeState loads the state stored in the session and verifies it matches
// the one returned by the provider, validating the provider too.
func (s *SSOService) consumeState(c *gin.Context, returned string) (*ssoState, error) {
	raw, ok := util.GetSession(c, ssoStateSessionKey).(string)
	if !ok || raw == "" {
		return nil, serializer.NewError(serializer.CodeCredentialInvalid, "Invalid or expired SSO state", nil)
	}

	record := &ssoState{}
	if err := json.Unmarshal([]byte(raw), record); err != nil {
		return nil, serializer.NewError(serializer.CodeCredentialInvalid, "Invalid SSO state", err)
	}

	if subtle.ConstantTimeCompare([]byte(record.State), []byte(returned)) != 1 {
		return nil, serializer.NewError(serializer.CodeCredentialInvalid, "SSO state mismatch", nil)
	}

	if record.Provider != s.provider {
		return nil, serializer.NewError(serializer.CodeCredentialInvalid, "SSO provider mismatch", nil)
	}

	util.DeleteSession(c, ssoStateSessionKey)
	return record, nil
}

// authorizeURL builds the provider authorize endpoint URL with the given state.
func (s *SSOService) authorizeURL(c *gin.Context, dep dependency.Dep, cfg *ssoConfig, state, challenge string) (string, error) {
	redirect, err := s.redirectURI(c, dep)
	if err != nil {
		return "", err
	}

	switch s.provider {
	case ProviderQQ:
		base := cfg.AuthURL
		if base == "" {
			base = "https://graph.qq.com/oauth2.0/authorize"
		}
		parsed, perr := url.Parse(base)
		if perr != nil {
			return "", serializer.NewError(serializer.CodeInternalSetting, "Invalid QQ authorize URL", perr)
		}
		q := parsed.Query()
		q.Set("response_type", "code")
		q.Set("client_id", cfg.AppID)
		q.Set("redirect_uri", redirect)
		q.Set("state", state)
		q.Set("scope", cfg.Scope)
		parsed.RawQuery = q.Encode()
		return parsed.String(), nil

	default:
		discovery, derr := s.discovery(c, dep, cfg.Endpoint)
		if derr != nil {
			return "", derr
		}
		parsed, perr := url.Parse(discovery.AuthorizationEndpoint)
		if perr != nil {
			return "", serializer.NewError(serializer.CodeInternalSetting, "Invalid authorization endpoint", perr)
		}
		q := parsed.Query()
		q.Set("response_type", "code")
		q.Set("client_id", cfg.AppID)
		q.Set("redirect_uri", redirect)
		q.Set("state", state)
		q.Set("scope", cfg.ScopePrepared())
		if challenge != "" {
			q.Set("code_challenge", challenge)
			q.Set("code_challenge_method", "S256")
		}
		parsed.RawQuery = q.Encode()
		return parsed.String(), nil
	}
}

// redirectURI returns the callback URL of this provider.
func (s *SSOService) redirectURI(c *gin.Context, dep dependency.Dep) (string, error) {
	base := dep.SettingProvider().SiteURL(c)
	if base == nil {
		return "", serializer.NewError(serializer.CodeInternalSetting, "Site URL is not configured", nil)
	}

	u := *base
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = ""
	u.RawQuery = ""
	u.Fragment = ""
	u.RawFragment = ""
	return fmt.Sprintf("%s/api/v4/session/sso/%s/callback", u.String(), string(s.provider)), nil
}

// discovery holds the fields of the OIDC discovery document we care about.
type discovery struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserInfoEndpoint      string `json:"userinfo_endpoint"`
}

// discovery fetches and returns the OIDC discovery document for the endpoint.
func (s *SSOService) discovery(c *gin.Context, dep dependency.Dep, endpoint string) (*discovery, error) {
	if strings.TrimSpace(endpoint) == "" {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "SSO endpoint is not configured", nil)
	}

	discoveryURL := strings.TrimRight(endpoint, "/") + "/.well-known/openid-configuration"
	client := dep.RequestClient(
		request.WithContext(c),
		request.WithLogger(logging.FromContext(c)),
	)
	resp := client.Request(http.MethodGet, discoveryURL, nil)
	if resp.Err != nil {
		return nil, serializer.NewError(serializer.CodeCallbackError, "Failed to fetch SSO discovery document: "+resp.Err.Error(), resp.Err)
	}
	raw, err := resp.GetResponse()
	if err != nil {
		return nil, serializer.NewError(serializer.CodeCallbackError, "Failed to read SSO discovery document", err)
	}

	d := &discovery{}
	if err := json.Unmarshal([]byte(raw), d); err != nil {
		return nil, serializer.NewError(serializer.CodeCallbackError, "Invalid SSO discovery document", err)
	}
	if d.AuthorizationEndpoint == "" || d.TokenEndpoint == "" || d.UserInfoEndpoint == "" {
		return nil, serializer.NewError(serializer.CodeCallbackError, "SSO discovery document is missing required endpoints", nil)
	}
	return d, nil
}

// exchangeCode exchanges the authorization code for tokens at the token endpoint.
func (s *SSOService) exchangeCode(c *gin.Context, dep dependency.Dep, cfg *ssoConfig, code, verifier, returnedState string) (string, error) {
	client := dep.RequestClient(
		request.WithContext(c),
		request.WithLogger(logging.FromContext(c)),
	)

	var tokenURL string
	switch s.provider {
	case ProviderQQ:
		tokenURL = cfg.TokenURL
		if tokenURL == "" {
			tokenURL = "https://graph.qq.com/oauth2.0/token"
		}
	default:
		discovery, err := s.discovery(c, dep, cfg.Endpoint)
		if err != nil {
			return "", err
		}
		tokenURL = discovery.TokenEndpoint
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", mustRedirectURI(s, c, dep))
	switch s.provider {
	case ProviderQQ:
		form.Set("client_id", cfg.AppID)
		form.Set("client_secret", cfg.AppSecret)
	default:
		form.Set("client_id", cfg.AppID)
		if cfg.AppSecret != "" {
			form.Set("client_secret", cfg.AppSecret)
		}
		if verifier != "" {
			form.Set("code_verifier", verifier)
		}
	}

	resp := client.Request(
		http.MethodPost, tokenURL, strings.NewReader(form.Encode()),
		request.WithHeader(http.Header{"Content-Type": {"application/x-www-form-urlencoded"}}),
	)
	if resp.Err != nil {
		return "", serializer.NewError(serializer.CodeCallbackError, "Failed to exchange SSO token: "+resp.Err.Error(), resp.Err)
	}
	raw, err := resp.GetResponse()
	if err != nil {
		return "", serializer.NewError(serializer.CodeCallbackError, "Failed to read SSO token response", err)
	}

	if s.provider == ProviderQQ {
		// QQ returns a query-string encoded token payload.
		token, terr := parseQQToken(raw)
		if terr != nil {
			return "", serializer.NewError(serializer.CodeCredentialInvalid, "Failed to parse QQ token response", terr)
		}
		return token, nil
	}

	tokenResp := struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
		TokenType   string `json:"token_type"`
	}{}
	if err := json.Unmarshal([]byte(raw), &tokenResp); err != nil || tokenResp.AccessToken == "" {
		return "", serializer.NewError(serializer.CodeCredentialInvalid, "Failed to parse SSO token response", err)
	}
	return tokenResp.AccessToken, nil
}

// mustRedirectURI is a best-effort helper used on paths already validated by authorizeURL.
func mustRedirectURI(s *SSOService, c *gin.Context, dep dependency.Dep) string {
	u, err := s.redirectURI(c, dep)
	if err != nil {
		return ""
	}
	return u
}

// parseQQToken parses the QQ token endpoint response which is form-encoded.
func parseQQToken(raw string) (string, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return "", err
	}
	token := values.Get("access_token")
	if token == "" {
		return "", errors.New("missing access_token in QQ response")
	}
	return token, nil
}

// fetchUserInfo obtains the unique identifier and a display nickname from the
// provider's userinfo. The identifier is the verified email when available,
// falling back to the OIDC `sub` (or QQ openid).
func (s *SSOService) fetchUserInfo(c *gin.Context, dep dependency.Dep, cfg *ssoConfig, accessToken string) (string, string, error) {
	client := dep.RequestClient(
		request.WithContext(c),
		request.WithLogger(logging.FromContext(c)),
	)

	var userInfoURL string
	switch s.provider {
	case ProviderQQ:
		userInfoURL = cfg.UserInfoURL
		if userInfoURL == "" {
			userInfoURL = "https://graph.qq.com/oauth2.0/me"
		}
	default:
		discovery, err := s.discovery(c, dep, cfg.Endpoint)
		if err != nil {
			return "", "", err
		}
		userInfoURL = discovery.UserInfoEndpoint
	}

	var resp *request.Response
	if s.provider == ProviderQQ {
		u, perr := url.Parse(userInfoURL)
		if perr != nil {
			return "", "", serializer.NewError(serializer.CodeInternalSetting, "Invalid QQ userinfo URL", perr)
		}
		q := u.Query()
		q.Set("access_token", accessToken)
		u.RawQuery = q.Encode()
		resp = client.Request(http.MethodGet, u.String(), nil)
	} else {
		resp = client.Request(
			http.MethodGet, userInfoURL, nil,
			request.WithHeader(http.Header{"Authorization": {"Bearer " + accessToken}}),
		)
	}
	if resp.Err != nil {
		return "", "", serializer.NewError(serializer.CodeCallbackError, "Failed to fetch SSO userinfo: "+resp.Err.Error(), resp.Err)
	}
	raw, err := resp.GetResponse()
	if err != nil {
		return "", "", serializer.NewError(serializer.CodeCallbackError, "Failed to read SSO userinfo", err)
	}

	if s.provider == ProviderQQ {
		payload := trimJSONPCallback(raw)
		if !strings.HasPrefix(payload, "{") {
			return "", "", serializer.NewError(serializer.CodeCredentialInvalid, "Failed to resolve QQ openid", nil)
		}
		p := struct {
			OpenID   string `json:"openid"`
			Nickname string `json:"nickname"`
		}{}
		if jsonErr := json.Unmarshal([]byte(payload), &p); jsonErr != nil || p.OpenID == "" {
			return "", "", serializer.NewError(serializer.CodeCredentialInvalid, "Failed to resolve QQ openid", nil)
		}
		return "qq:" + p.OpenID, firstNonEmpty(p.Nickname, "QQ User"), nil
	}

	p := struct {
		Sub      string `json:"sub"`
		Email    string `json:"email"`
		Verified *bool  `json:"email_verified"`
		Name     string `json:"name"`
		Nick     string `json:"nickname"`
	}{}
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return "", "", serializer.NewError(serializer.CodeCredentialInvalid, "Invalid SSO userinfo payload", err)
	}

	// Prefer a verified email, otherwise fall back to the opaque subject.
	if p.Email != "" && (p.Verified == nil || *p.Verified) {
		return strings.ToLower(p.Email), firstNonEmpty(p.Name, p.Nick, p.Email), nil
	}
	if p.Sub != "" {
		return "sub:" + p.Sub, firstNonEmpty(p.Name, p.Nick, "SSO User"), nil
	}
	return "", "", serializer.NewError(serializer.CodeCredentialInvalid, "SSO userinfo is missing email and sub", nil)
}

// trimJSONPCallback strips the `callback( ... );` wrapper QQ sometimes returns.
func trimJSONPCallback(raw string) string {
	s := strings.TrimSpace(raw)
	if idx := strings.Index(s, "{"); idx >= 0 {
		s = s[idx:]
	}
	if idx := strings.Index(s, "}"); idx >= 0 {
		s = s[:idx+1]
	}
	return s
}

// findOrCreateUser looks up an account by the SSO identifier and provisions a
// new one (with the default group) when it does not exist yet.
func (s *SSOService) findOrCreateUser(c *gin.Context, dep dependency.Dep, identifier, nickname string) (*ent.User, error) {
	userClient := dep.UserClient()

	var u *ent.User
	if !strings.HasPrefix(identifier, "sub:") && !strings.HasPrefix(identifier, "qq:") {
		u, _ = userClient.GetByEmail(c, identifier)
	}

	if u == nil {
		email := identifier
		if strings.HasPrefix(identifier, "sub:") || strings.HasPrefix(identifier, "qq:") {
			email = ""
		}
		if email == "" {
			return nil, serializer.NewError(serializer.CodeUserNotFound, "SSO did not provide an email, cannot create an account", nil)
		}

		args := &inventory.NewUserArgs{
			Email:   email,
			Nick:    nickname,
			Status:  user.StatusActive,
			GroupID: dep.SettingProvider().DefaultGroup(c),
		}
		created, err := userClient.Create(c, args)
		if err != nil {
			if err == inventory.ErrUserEmailExisted {
				u, _ = userClient.GetByEmail(c, email)
			} else {
				return nil, serializer.NewError(serializer.CodeDBError, "Failed to provision SSO account", err)
			}
		}
		if created != nil {
			u = created
		}
	}
	if u == nil {
		return nil, serializer.NewError(serializer.CodeUserNotFound, "SSO account could not be resolved", nil)
	}

	switch u.Status {
	case user.StatusActive:
		return u, nil
	case user.StatusInactive:
		return nil, serializer.NewError(serializer.CodeUserNotActivated, "This account is not activated", nil)
	default:
		return nil, serializer.NewError(serializer.CodeUserBaned, "This account has been blocked", nil)
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
