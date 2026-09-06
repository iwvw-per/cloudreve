package callback

import (
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/service/pro"
	"github.com/gin-gonic/gin"
)

// PaymentService 处理各支付渠道的异步通知回调。
type PaymentService struct{}

// NotifyResult 回调处理结果，Controller 据此返回对应渠道约定的应答正文。
type NotifyResult struct {
	ReplyXML bool
	Reply    string
}

// payment settings keys
const (
	settingAlipayAppID      = "alipay_appid"
	settingAlipayPublicKey  = "alipay_public_key"
	settingWechatAppID      = "wechat_appid"
	settingWechatMchID      = "wechat_mch_id"
	settingWechatAPIKey     = "wechat_api_key"
	settingPayJSKey         = "payjs_key"
	settingCustomPaymentKey = "custom_payment_key"
	wechatSignMD5           = "MD5"
	wechatSignHmacSHA256    = "HMAC-SHA256"
)

// verifiedNotify 校验通过后抽离出的通用支付通知字段。
type verifiedNotify struct {
	OrderNo  string
	Amount   int // 单位：分
	Success  bool
	Provider types.PaymentProvider
}

// HandleAlipay 处理支付宝当面付异步通知（RSA2 验签，form-urlencoded 回调）。
func (s *PaymentService) HandleAlipay(c *gin.Context) NotifyResult {
	dep := dependency.FromContext(c)
	params := formParams(c)

	ok, notify, err := s.verifyAlipay(dep, c, params)
	if err != nil {
		return failResult(err)
	}
	if !ok {
		return failResult(fmt.Errorf("alipay signature verification failed"))
	}

	// 非支付成功状态（如等待付款）按约定应答 success，但不履约，避免重复通知。
	if !notify.Success {
		return successResult(false)
	}

	if err := s.processPaid(c, dep, notify); err != nil {
		return failResult(err)
	}
	return successResult(false)
}

// HandleWechat 处理微信直连 Native 扫码异步通知（XML 回调，MD5/HMAC-SHA256 验签）。
func (s *PaymentService) HandleWechat(c *gin.Context) NotifyResult {
	dep := dependency.FromContext(c)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return wechatFailReply()
	}
	params, err := xmlToMap(body)
	if err != nil {
		return wechatFailReply()
	}

	ok, notify, err := s.verifyWechat(dep, params)
	if err != nil {
		return wechatFailReply()
	}
	if !ok {
		return wechatFailReply()
	}
	if !notify.Success {
		return successResult(true)
	}

	if err := s.processPaid(c, dep, notify); err != nil {
		return wechatFailReply()
	}
	return successResult(true)
}

// HandlePayJS 处理 PAYJS 异步通知（MD5 验签，form-urlencoded 回调）。
func (s *PaymentService) HandlePayJS(c *gin.Context) NotifyResult {
	dep := dependency.FromContext(c)
	params := formParams(c)

	ok, notify, err := s.verifyPayJS(dep, params)
	if err != nil {
		return failResult(err)
	}
	if !ok {
		return failResult(fmt.Errorf("payjs signature verification failed"))
	}
	if !notify.Success {
		return successResult(false)
	}

	if err := s.processPaid(c, dep, notify); err != nil {
		return failResult(err)
	}
	return successResult(false)
}

// HandleCustom 处理自定义支付渠道异步通知（HMAC-SHA256 验签，form-urlencoded 回调）。
func (s *PaymentService) HandleCustom(c *gin.Context) NotifyResult {
	dep := dependency.FromContext(c)
	params := formParams(c)

	ok, notify, err := s.verifyCustom(dep, params)
	if err != nil {
		return failResult(err)
	}
	if !ok {
		return failResult(fmt.Errorf("custom payment signature verification failed"))
	}
	if !notify.Success {
		return successResult(false)
	}

	if err := s.processPaid(c, dep, notify); err != nil {
		return failResult(err)
	}
	return successResult(false)
}

func successResult(xmlReply bool) NotifyResult {
	if xmlReply {
		return NotifyResult{ReplyXML: true, Reply: "<xml><return_code><![CDATA[SUCCESS]]></return_code></xml>"}
	}
	return NotifyResult{Reply: "success"}
}

func failResult(err error) NotifyResult {
	return NotifyResult{Reply: "fail"}
}

func wechatFailReply() NotifyResult {
	return NotifyResult{ReplyXML: true, Reply: "<xml><return_code><![CDATA[FAIL]]></return_code><return_msg><![CDATA[FAIL]]></return_msg></xml>"}
}

// verifyAlipay 使用支付宝公钥对回调参数做 RSA2(SHA256) 验签。
func (s *PaymentService) verifyAlipay(dep dependency.Dep, ctx context.Context, params map[string]string) (bool, verifiedNotify, error) {
	pubKey := settingValue(ctx, dep, settingAlipayPublicKey, "")
	if pubKey == "" {
		return false, verifiedNotify{}, fmt.Errorf("alipay_public_key is not configured")
	}

	sign := params["sign"]
	if sign == "" {
		return false, verifiedNotify{}, fmt.Errorf("missing alipay sign")
	}

	content := sortedSignContent(params, map[string]struct{}{"sign": {}, "sign_type": {}})
	if !rsaVerifySHA256(content, sign, pubKey) {
		return false, verifiedNotify{}, fmt.Errorf("invalid alipay signature")
	}

	amount, err := yuanToCents(params["total_amount"])
	if err != nil {
		return false, verifiedNotify{}, fmt.Errorf("invalid alipay total_amount: %w", err)
	}

	status := params["trade_status"]
	return true, verifiedNotify{
		OrderNo:  params["out_trade_no"],
		Amount:   amount,
		Success:  status == "TRADE_SUCCESS" || status == "TRADE_FINISHED",
		Provider: types.PaymentProviderAlipay,
	}, nil
}

// verifyWechat 使用商户密钥对回调参数做 MD5 或 HMAC-SHA256 验签。
func (s *PaymentService) verifyWechat(dep dependency.Dep, params map[string]string) (bool, verifiedNotify, error) {
	apiKey := settingValue(context.Background(), dep, settingWechatAPIKey, "")
	if apiKey == "" {
		return false, verifiedNotify{}, fmt.Errorf("wechat_api_key is not configured")
	}

	sign := params["sign"]
	if sign == "" {
		return false, verifiedNotify{}, fmt.Errorf("missing wechat sign")
	}

	signType := params["sign_type"]
	if signType == "" {
		signType = wechatSignMD5
	}

	content := sortedSignContent(params, map[string]struct{}{"sign": {}})
	if !signMatch(content, sign, signType, apiKey) {
		return false, verifiedNotify{}, fmt.Errorf("invalid wechat signature")
	}

	fee, err := strconv.Atoi(params["total_fee"])
	if err != nil {
		return false, verifiedNotify{}, fmt.Errorf("invalid wechat total_fee: %w", err)
	}

	success := params["return_code"] == "SUCCESS" && params["result_code"] == "SUCCESS"
	return true, verifiedNotify{
		OrderNo:  params["out_trade_no"],
		Amount:   fee,
		Success:  success,
		Provider: types.PaymentProviderWechat,
	}, nil
}

// verifyPayJS 使用 PAYJS 商户密钥对回调做 MD5 验签。
// PAYJS 签名规则：sign = md5(out_trade_no + total_fee + key) 的大写形式。
func (s *PaymentService) verifyPayJS(dep dependency.Dep, params map[string]string) (bool, verifiedNotify, error) {
	key := settingValue(context.Background(), dep, settingPayJSKey, "")
	if key == "" {
		return false, verifiedNotify{}, fmt.Errorf("payjs_key is not configured")
	}

	orderNo := params["out_trade_no"]
	fee := params["total_fee"]
	expected := strings.ToUpper(md5Hex(orderNo + fee + key))
	if params["sign"] == "" || !strings.EqualFold(params["sign"], expected) {
		return false, verifiedNotify{}, fmt.Errorf("invalid payjs signature")
	}

	amount, err := strconv.Atoi(fee)
	if err != nil {
		return false, verifiedNotify{}, fmt.Errorf("invalid payjs total_fee: %w", err)
	}

	return true, verifiedNotify{
		OrderNo:  orderNo,
		Amount:   amount,
		Success:  params["return_code"] == "1",
		Provider: types.PaymentProviderPayJS,
	}, nil
}

// verifyCustom 自定义渠道验签：sign = hmac-sha256(order_no|amount|key) 的 hex 小写。
func (s *PaymentService) verifyCustom(dep dependency.Dep, params map[string]string) (bool, verifiedNotify, error) {
	key := settingValue(context.Background(), dep, settingCustomPaymentKey, "")
	if key == "" {
		return false, verifiedNotify{}, fmt.Errorf("custom_payment_key is not configured")
	}

	orderNo := params["order_no"]
	amountStr := params["amount"]
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(orderNo + "|" + amountStr))
	expected := hex.EncodeToString(mac.Sum(nil))
	if params["sign"] == "" || !strings.EqualFold(params["sign"], expected) {
		return false, verifiedNotify{}, fmt.Errorf("invalid custom payment signature")
	}

	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		return false, verifiedNotify{}, fmt.Errorf("invalid custom payment amount: %w", err)
	}

	return true, verifiedNotify{
		OrderNo:  orderNo,
		Amount:   amount,
		Success:  params["status"] == "success",
		Provider: types.PaymentProviderCustom,
	}, nil
}

// processPaid 通用回调履约流程：置为 paid -> 按商品履约 -> 置为 fulfilled。
func (s *PaymentService) processPaid(c *gin.Context, dep dependency.Dep, notify verifiedNotify) error {
	orderClient := dep.OrderClient()
	userClient := dep.UserClient()
	eventClient := dep.EventClient()

	// 开启事务，保证订单状态流转与履约原子生效。
	txOrder, tx, ctx, err := inventory.WithTx(c, orderClient)
	if err != nil {
		return fmt.Errorf("failed to create payment transaction: %w", err)
	}
	txProduct, _ := inventory.InheritTx(ctx, dep.ProductClient())
	txUser, _ := inventory.InheritTx(ctx, userClient)
	txEvent, _ := inventory.InheritTx(ctx, eventClient)

	order, err := txOrder.GetByOrderNo(ctx, notify.OrderNo)
	if err != nil {
		inventory.Rollback(tx)
		return fmt.Errorf("order %s not found: %w", notify.OrderNo, err)
	}

	if order.Amount != notify.Amount {
		inventory.Rollback(tx)
		return fmt.Errorf("amount mismatch for order %s: expected %d, got %d", notify.OrderNo, order.Amount, notify.Amount)
	}

	// 幂等：已支付或已履约的订单直接视为成功，避免重复到账。
	curStatus := types.OrderStatus(order.Status)
	if curStatus == types.OrderStatusPaid || curStatus == types.OrderStatusFulfilled {
		inventory.Commit(tx)
		return nil
	}

	if _, err := txOrder.UpdateStatus(ctx, order.ID, types.OrderStatusPaid, notify.Provider); err != nil {
		inventory.Rollback(tx)
		return fmt.Errorf("failed to mark order %s paid: %w", notify.OrderNo, err)
	}
	writePaidEvent(ctx, txEvent, order)

	if err := pro.FulfillOrder(ctx, dep, txProduct, txUser, txEvent, order, notify.Provider); err != nil {
		writeFulfillFailedEvent(ctx, txEvent, order, err)
		inventory.Rollback(tx)
		return err
	}

	if _, err := txOrder.MarkFulfilled(ctx, order.ID); err != nil {
		inventory.Rollback(tx)
		return fmt.Errorf("failed to mark order %s fulfilled: %w", notify.OrderNo, err)
	}

	if err := inventory.Commit(tx); err != nil {
		return fmt.Errorf("failed to commit payment transaction: %w", err)
	}
	return nil
}

// ---------- 审计事件写入 ----------

func writePaidEvent(ctx context.Context, eventClient inventory.EventClient, order *ent.Order) {
	_, _ = eventClient.Create(ctx, &inventory.CreateEventParams{
		Type:   types.AuditTypePaymentPaid,
		UserID: order.UserOrders,
		Content: &types.AuditContent{
			PaymentID: order.ID,
			Sku:       order.OrderNo,
		},
	})
}

func writeFulfillFailedEvent(ctx context.Context, eventClient inventory.EventClient, order *ent.Order, fulfillErr error) {
	_, _ = eventClient.Create(ctx, &inventory.CreateEventParams{
		Type:   types.AuditTypePaymentFulfillFailed,
		UserID: order.UserOrders,
		Content: &types.AuditContent{
			PaymentID: order.ID,
			Error:     fulfillErr.Error(),
			Failed:    true,
		},
	})
}

// ---------- 通用签名工具 ----------

func settingValue(ctx context.Context, dep dependency.Dep, key, def string) string {
	v, err := dep.SettingClient().Get(ctx, key)
	if err != nil {
		return def
	}
	return v
}

// formParams 提取 form-urlencoded 回调参数。
func formParams(c *gin.Context) map[string]string {
	_ = c.Request.ParseForm()
	params := make(map[string]string, len(c.Request.PostForm))
	for k, vs := range c.Request.PostForm {
		if len(vs) > 0 {
			params[k] = vs[0]
		}
	}
	return params
}

// sortedSignContent 将除 exclude 外的参数按 key 排序后拼接为 k=v&... 形式。
func sortedSignContent(params map[string]string, exclude map[string]struct{}) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if _, skip := exclude[k]; skip {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	return strings.Join(parts, "&")
}

// signMatch 校验 MD5 或 HMAC-SHA256 签名（大小写不敏感）。
// key 仅用于 HMAC-SHA256 的 HMAC 密钥；MD5 模式下 key 会拼接到 content 尾部。
func signMatch(content, sign, signType, key string) bool {
	var actual string
	if signType == wechatSignHmacSHA256 {
		mac := hmac.New(sha256.New, []byte(key))
		mac.Write([]byte(content))
		actual = strings.ToUpper(hex.EncodeToString(mac.Sum(nil)))
	} else {
		actual = strings.ToUpper(md5Hex(content + "&key=" + key))
	}
	return strings.EqualFold(actual, sign)
}

func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

// rsaVerifySHA256 使用 PEM 编码的 RSA 公钥验证 RSA2(SHA256withRSA) 签名。
func rsaVerifySHA256(content, sign, publicKeyPEM string) bool {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return false
	}
	pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return false
	}
	pub, ok := pubAny.(*rsa.PublicKey)
	if !ok {
		return false
	}
	sig, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return false
	}
	digest := sha256.Sum256([]byte(content))
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], sig) == nil
}

// yuanToCents 将支付宝元金额字符串转换为分。
func yuanToCents(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("empty amount")
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return int(math.Round(f * 100)), nil
}

// xmlToMap 将微信回调 XML 解析为 map。
func xmlToMap(data []byte) (map[string]string, error) {
	params := make(map[string]string)
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	var key string
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			key = t.Name.Local
		case xml.CharData:
			if key != "" {
				if v := strings.TrimSpace(string(t)); v != "" {
					params[key] = v
				}
			}
		case xml.EndElement:
			key = ""
		}
	}
	return params, nil
}
