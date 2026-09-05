package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/task"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager/entitysource"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gofrs/uuid"
	"github.com/samber/lo"
	"golang.org/x/tools/container/intsets"
)

type (
	RelocateTask struct {
		*queue.DBTask

		l        logging.Logger
		state    *RelocateTaskState
		progress queue.Progresses
	}

	RelocateTaskPhase string

	RelocateTaskState struct {
		SrcURI         string             `json:"src_uri,omitempty"`
		TargetPolicyID int                `json:"target_policy_id,omitempty"`
		Recursive      bool               `json:"recursive,omitempty"`
		Phase          RelocateTaskPhase  `json:"phase,omitempty"`
		Uris           []string           `json:"uris,omitempty"`
		Processed      int                `json:"processed,omitempty"`
		Total          int                `json:"total,omitempty"`
		Failed         int                `json:"failed,omitempty"`
		Errors         []string           `json:"errors,omitempty"`
	}
)

const (
	// RelocateTaskPhaseTransfer is the phase when files are being transferred
	// to the target storage policy. The empty string phase is used while the
	// source path is being scanned.
	RelocateTaskPhaseTransfer RelocateTaskPhase = "transfer"

	// ProgressTypeRelocateCount aligns with the frontend ProgressKeys.relocate.
	ProgressTypeRelocateCount = "relocate"
	ProgressTypeRelocateSize  = "relocate_size"

	RelocateBatchSize = 10
	// RelocateMaxErrors caps the number of recorded per-file errors kept in task
	// private state, so the task state does not grow unboundedly.
	RelocateMaxErrors = 100
)

func init() {
	queue.RegisterResumableTaskFactory(queue.RelocateTaskType, NewRelocateTaskFromModel)
}

// NewRelocateTask creates a new RelocateTask.
func NewRelocateTask(ctx context.Context, srcUri string, targetPolicyID int, recursive bool) (queue.Task, error) {
	state := &RelocateTaskState{
		SrcURI:         srcUri,
		TargetPolicyID: targetPolicyID,
		Recursive:      recursive,
	}
	stateBytes, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}

	t := &RelocateTask{
		DBTask: &queue.DBTask{
			Task: &ent.Task{
				Type:          queue.RelocateTaskType,
				CorrelationID: logging.CorrelationID(ctx),
				PrivateState:  string(stateBytes),
				PublicState:   &types.TaskPublicState{},
			},
			DirectOwner: inventory.UserFromContext(ctx),
		},
	}
	return t, nil
}

func NewRelocateTaskFromModel(t *ent.Task) queue.Task {
	return &RelocateTask{
		DBTask: &queue.DBTask{
			Task: t,
		},
	}
}

func (m *RelocateTask) Do(ctx context.Context) (task.Status, error) {
	dep := dependency.FromContext(ctx)
	m.l = dep.Logger()

	// unmarshal state
	state := &RelocateTaskState{}
	if err := json.Unmarshal([]byte(m.State()), state); err != nil {
		return task.StatusError, fmt.Errorf("failed to unmarshal state: %w", err)
	}
	m.state = state

	// initialize progress
	m.Lock()
	if m.progress == nil {
		m.progress = make(queue.Progresses)
	}
	if m.progress[ProgressTypeRelocateCount] == nil {
		m.progress[ProgressTypeRelocateCount] = &queue.Progress{}
	}
	if m.progress[ProgressTypeRelocateSize] == nil {
		m.progress[ProgressTypeRelocateSize] = &queue.Progress{}
	}
	atomic.StoreInt64(&m.progress[ProgressTypeRelocateCount].Total, int64(m.state.Total))
	atomic.StoreInt64(&m.progress[ProgressTypeRelocateCount].Current, int64(m.state.Processed))
	m.Unlock()

	var (
		next = task.StatusCompleted
		err  error
	)
	switch m.state.Phase {
	case "":
		next, err = m.scan(ctx, dep)
	case RelocateTaskPhaseTransfer:
		next, err = m.relocate(ctx, dep)
	default:
		next, err = task.StatusError, fmt.Errorf("unknown phase %q: %w", m.state.Phase, queue.CriticalErr)
	}

	newStateStr, marshalErr := json.Marshal(m.state)
	if marshalErr != nil {
		return task.StatusError, fmt.Errorf("failed to marshal state: %w", marshalErr)
	}

	m.Lock()
	m.Task.PrivateState = string(newStateStr)
	m.Unlock()
	return next, err
}

// scan walks the source path and collects all file URIs that need to be
// relocated, then moves the task into the relocate phase.
func (m *RelocateTask) scan(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	uri, err := fs.NewUriFromString(m.state.SrcURI)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to parse src uri %q: %s (%w)", m.state.SrcURI, err, queue.CriticalErr)
	}

	user := inventory.UserFromContext(ctx)
	fm := manager.NewFileManager(dep, user)
	defer fm.Recycle()

	depth := 1
	if m.state.Recursive {
		depth = intsets.MaxInt
	}

	fileUris := make([]string, 0)
	err = fm.Walk(ctx, uri, depth, func(file fs.File, level int) error {
		if file.Type() == types.FileTypeFile {
			fileUris = append(fileUris, file.Uri(false).String())
		}
		return nil
	}, dbfs.WithRequiredCapabilities(dbfs.NavigatorCapabilityDownloadFile))
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to scan files under %q: %s (%w)", m.state.SrcURI, err, queue.CriticalErr)
	}

	m.state.Uris = fileUris
	m.state.Total = len(fileUris)
	m.state.Processed = 0
	m.state.Failed = 0
	m.state.Errors = nil
	m.state.Phase = RelocateTaskPhaseTransfer

	m.l.Info("Relocate scan finished, %d file(s) to be relocated.", len(fileUris))

	m.Lock()
	m.progress[ProgressTypeRelocateCount] = &queue.Progress{Total: int64(len(fileUris))}
	m.progress[ProgressTypeRelocateSize] = &queue.Progress{}
	m.Unlock()

	m.ResumeAfter(0)
	return task.StatusSuspending, nil
}

// relocate processes the collected files in batches until all of them are done.
func (m *RelocateTask) relocate(ctx context.Context, dep dependency.Dep) (task.Status, error) {
	if len(m.state.Uris) == 0 {
		m.l.Info("No file to relocate, task completed.")
		return task.StatusCompleted, nil
	}

	targetPolicy, err := dep.StoragePolicyClient().GetPolicyByID(ctx, m.state.TargetPolicyID)
	if err != nil {
		return task.StatusError, fmt.Errorf("failed to get target storage policy %d: %s (%w)", m.state.TargetPolicyID, err, queue.CriticalErr)
	}

	user := inventory.UserFromContext(ctx)
	fm := manager.NewFileManager(dep, user)
	defer fm.Recycle()

	end := m.state.Processed + RelocateBatchSize
	if end > len(m.state.Uris) {
		end = len(m.state.Uris)
	}

	for _, uriStr := range m.state.Uris[m.state.Processed:end] {
		select {
		case <-ctx.Done():
			return task.StatusError, ctx.Err()
		default:
		}

		uri, err := fs.NewUriFromString(uriStr)
		if err != nil {
			m.l.Warning("Failed to parse uri %q: %s", uriStr, err)
			m.state.Failed++
			m.recordError(uriStr, err)
		} else if err := m.migrateFile(ctx, dep, fm, uri, targetPolicy); err != nil {
			m.l.Warning("Failed to relocate %q: %s", uriStr, err)
			m.state.Failed++
			m.recordError(uriStr, err)
		}

		m.state.Processed++
		atomic.AddInt64(&m.progress[ProgressTypeRelocateCount].Current, 1)
	}

	if m.state.Processed >= len(m.state.Uris) {
		// 迁移全部完成后，若源路径是目录，将目录的偏好存储策略更新为目标策略，
		// 使目录列表顶部展示的存储策略标签与新策略一致。
		if err := m.updateSourceDirPreference(ctx, dep); err != nil {
			m.l.Warning("Failed to update source directory preferred policy: %s", err)
		}
		m.l.Info("Relocate finished, %d file(s) migrated, %d failed.",
			len(m.state.Uris)-m.state.Failed, m.state.Failed)
		return task.StatusCompleted, nil
	}

	m.ResumeAfter(0)
	return task.StatusSuspending, nil
}

// recordError appends an error to the task state, capped at RelocateMaxErrors.
// updateSourceDirPreference updates the preferred storage policy of the source
// directory (if the relocation target is a directory) so that the file manager
// top indicator reflects the new storage policy after relocation completes.
func (m *RelocateTask) updateSourceDirPreference(ctx context.Context, dep dependency.Dep) error {
	uri, err := fs.NewUriFromString(m.state.SrcURI)
	if err != nil {
		return fmt.Errorf("failed to parse src uri %q: %w", m.state.SrcURI, err)
	}

	user := inventory.UserFromContext(ctx)
	fm := manager.NewFileManager(dep, user)
	defer fm.Recycle()

	srcFile, err := fm.Get(ctx, uri, dbfs.WithNotRoot())
	if err != nil {
		return fmt.Errorf("failed to get source file %q: %w", m.state.SrcURI, err)
	}
	if srcFile == nil {
		return fmt.Errorf("source file %q is nil", m.state.SrcURI)
	}
	m.l.Info("Relocate preference: srcFile type=%d folder=%d", srcFile.Type(), types.FileTypeFolder)
	if srcFile.Type() != types.FileTypeFolder {
		// Only directories get their preferred policy updated.
		return nil
	}

	dbf, ok := srcFile.(*dbfs.File)
	if !ok || dbf.Model == nil {
		return fmt.Errorf("unsupported file type for preferred policy update")
	}

	props := dbf.Model.Props
	if props == nil {
		props = &types.FileProps{}
	}
	props.PreferredStoragePolicyID = m.state.TargetPolicyID
	if _, err := dep.FileClient().UpdateProps(ctx, dbf.Model, props); err != nil {
		return fmt.Errorf("failed to update preferred storage policy of %q: %w", m.state.SrcURI, err)
	}

	m.l.Info("Relocate preference updated for %q to policy %d", m.state.SrcURI, m.state.TargetPolicyID)
	return nil
}

func (m *RelocateTask) recordError(uri string, err error) {
	m.state.Errors = append(m.state.Errors, fmt.Sprintf("%s: %s", uri, err))
	if len(m.state.Errors) > RelocateMaxErrors {
		m.state.Errors = m.state.Errors[len(m.state.Errors)-RelocateMaxErrors:]
	}
}

// migrateFile relocates a single file's primary entity to the target storage
// policy. It is non-invasive: the data is first uploaded to the target policy
// and only after that succeeds are the file's primary entity reference and
// storage policy switched; the source entity is then unlinked so the entity
// recycle routine can clean it up later.
func (m *RelocateTask) migrateFile(ctx context.Context, dep dependency.Dep, fm manager.FileManager, uri *fs.URI, targetPolicy *ent.StoragePolicy) error {
	file, err := fm.Get(ctx, uri, dbfs.WithFileEntities(), dbfs.WithNotRoot())
	if err != nil {
		return fmt.Errorf("failed to get file %q: %w", uri, err)
	}
	if file.Type() != types.FileTypeFile {
		return nil
	}

	entity := file.PrimaryEntity()
	if entity == nil || entity.ID() == 0 {
		m.l.Debug("File %q has no primary entity, skipping.", uri)
		return nil
	}
	if entity.PolicyID() == targetPolicy.ID {
		m.l.Debug("File %q is already on target storage policy, skipping.", uri)
		return nil
	}

	user := inventory.UserFromContext(ctx)

	fc := dep.FileClient()
	fileModel, err := fc.GetByID(ctx, file.ID())
	if err != nil {
		return fmt.Errorf("failed to get file model for %q: %w", uri, err)
	}

	// If a previous partial run already uploaded an entity to the target policy,
	// reuse it instead of uploading again.
	reused, err := m.reuseRelocatedEntity(ctx, fc, file, entity, targetPolicy, user, fileModel)
	if err != nil {
		return err
	}
	if reused {
		return nil
	}

	// 1. Read the source entity data.
	es, err := fm.GetEntitySource(ctx, 0, fs.WithEntity(entity))
	if err != nil {
		return fmt.Errorf("failed to get entity source for %q: %w", uri, err)
	}
	es.Apply(entitysource.WithContext(ctx))

	// 2. Generate the physical save path on the target policy.
	savePath := generateRelocateSavePath(targetPolicy, file.Name(), uri.Dir(), user.ID)

	// 3. Generate encryption metadata if the target policy requires it.
	var encryptMetadata *types.EncryptMetadata
	if targetPolicy.Settings != nil && targetPolicy.Settings.Encryption {
		encryptor, err := dep.EncryptorFactory(ctx)(types.CipherAES256CTR)
		if err != nil {
			es.Close()
			return fmt.Errorf("failed to create cryptor for %q: %w", uri, err)
		}
		encryptMetadata, err = encryptor.GenerateMetadata(ctx)
		if err != nil {
			es.Close()
			return fmt.Errorf("failed to generate encrypt metadata for %q: %w", uri, err)
		}
	}

	// 4. Upload the data to the target storage policy.
	uploadReq := &fs.UploadRequest{
		Props: &fs.UploadProps{
			Uri:             uri,
			Size:            entity.Size(),
			SavePath:        savePath,
			UploadSessionID: uuid.Must(uuid.NewV4()).String(),
		},
		File:   es,
		Seeker: es,
		ProgressFunc: func(current, diff int64, total int64) {
			atomic.AddInt64(&m.progress[ProgressTypeRelocateSize].Current, diff)
		},
	}
	if encryptMetadata != nil {
		session := &fs.UploadSession{
			Policy:          targetPolicy,
			EncryptMetadata: encryptMetadata,
			Props:           uploadReq.Props,
		}
		err = fm.Upload(ctx, uploadReq, targetPolicy, session)
	} else {
		var d driver.Handler
		d, err = fm.GetStorageDriver(ctx, targetPolicy)
		if err == nil {
			err = d.Put(ctx, uploadReq)
		}
	}
	if err != nil {
		es.Close()
		return fmt.Errorf("failed to upload file %q to target storage policy: %w", uri, err)
	}

	// 5. Record the new entity in DB. Only after this point does the file begin
	// to reference the target policy, so a failure here leaves the source intact.
	newEntity, diff, err := fc.CreateEntity(ctx, fileModel, &inventory.EntityParameters{
		EntityType:      types.EntityTypeVersion,
		StoragePolicyID: targetPolicy.ID,
		Source:          savePath,
		Size:            entity.Size(),
		ModifiedAt:      lo.ToPtr(entity.UpdatedAt()),
		UploadSessionID: uuid.Must(uuid.NewV4()),
		Importing:       true,
		EncryptMetadata: encryptMetadata,
	})
	if err != nil {
		rollbackRelocateUpload(ctx, m.l, fm, targetPolicy, savePath)
		return fmt.Errorf("failed to create relocated entity for %q: %w", uri, err)
	}
	if err := dep.UserClient().ApplyStorageDiff(ctx, diff); err != nil {
		m.l.Warning("Failed to apply storage diff for relocated entity %d: %s", newEntity.ID, err)
	}

	// 6. Switch the file's primary entity reference.
	if err := fc.SetPrimaryEntity(ctx, fileModel, newEntity); err != nil {
		m.rollbackCreatedEntity(ctx, dep, fm, fileModel, newEntity, savePath, targetPolicy, user)
		return fmt.Errorf("failed to switch primary entity for %q: %w", uri, err)
	}

	// 7. Update the file's storage policy field.
	if err := fc.GetClient().File.UpdateOne(fileModel).SetStoragePoliciesID(targetPolicy.ID).Exec(ctx); err != nil {
		m.l.Warning("Failed to update storage policy field of file %q: %s", uri, err)
	}

	// 8. Unlink the old entity. Its reference count drops to zero and the
	// entity recycle routine will remove the source data afterwards.
	if _, err := fc.UnlinkEntity(ctx, entity.Model(), fileModel, user); err != nil {
		m.l.Warning("Failed to unlink old entity of %q: %s", uri, err)
	}

	return nil
}

// reuseRelocatedEntity promotes an entity that a previous partial run already
// uploaded to the target policy, so the migration is idempotent across retries.
func (m *RelocateTask) reuseRelocatedEntity(ctx context.Context, fc inventory.FileClient,
	file fs.File, entity fs.Entity, targetPolicy *ent.StoragePolicy, user *ent.User, fileModel *ent.File) (bool, error) {
	for _, e := range file.Entities() {
		if e.ID() == entity.ID() || e.Type() != types.EntityTypeVersion {
			continue
		}
		if e.PolicyID() != targetPolicy.ID || e.Size() != entity.Size() {
			continue
		}

		if err := fc.SetPrimaryEntity(ctx, fileModel, e.Model()); err != nil {
			return false, fmt.Errorf("failed to reuse relocated entity for %q: %w", file.Uri(false), err)
		}
		if err := fc.GetClient().File.UpdateOne(fileModel).SetStoragePoliciesID(targetPolicy.ID).Exec(ctx); err != nil {
			m.l.Warning("Failed to update storage policy field of file %q: %s", file.Uri(false), err)
		}
		if _, err := fc.UnlinkEntity(ctx, entity.Model(), fileModel, user); err != nil {
			m.l.Warning("Failed to unlink old entity of %q: %s", file.Uri(false), err)
		}

		m.l.Info("Reused existing entity %d for %q.", e.ID(), file.Uri(false))
		return true, nil
	}

	return false, nil
}

// rollbackCreatedEntity removes the entity record and uploaded file when the
// primary entity switch fails, keeping the file unchanged.
func (m *RelocateTask) rollbackCreatedEntity(ctx context.Context, dep dependency.Dep, fm manager.FileManager,
	file *ent.File, entity *ent.Entity, savePath string, targetPolicy *ent.StoragePolicy, owner *ent.User) {
	fc := dep.FileClient()
	if diff, err := fc.UnlinkEntity(ctx, entity, file, owner); err != nil {
		m.l.Warning("Failed to unlink created entity %d: %s", entity.ID, err)
	} else if err := dep.UserClient().ApplyStorageDiff(ctx, diff); err != nil {
		m.l.Warning("Failed to apply storage diff during rollback: %s", err)
	}
	rollbackRelocateUpload(ctx, m.l, fm, targetPolicy, savePath)
}

// rollbackRelocateUpload deletes the physical file uploaded to the target
// policy when the migration failed before the DB reference was switched.
func rollbackRelocateUpload(ctx context.Context, l logging.Logger, fm manager.FileManager, policy *ent.StoragePolicy, savePath string) {
	d, err := fm.GetStorageDriver(ctx, policy)
	if err != nil {
		return
	}
	if failed, err := d.Delete(ctx, savePath); err != nil {
		l.Warning("Failed to clean up uploaded file %q: %s, failed: %v", savePath, err, failed)
	}
}

// generateRelocateSavePath generates the physical save path for a relocated
// entity based on the target storage policy's directory/file name rules.
func generateRelocateSavePath(policy *ent.StoragePolicy, name, dir string, userID int) string {
	currentTime := time.Now()
	dynamicReplace := func(rule string, pathAvailable bool) string {
		return util.ReplaceMagicVar(rule, fs.Separator, pathAvailable, false, currentTime, userID, name, dir, "")
	}

	dirRule := policy.DirNameRule
	dirRule = filepath.ToSlash(dirRule)
	dirRule = dynamicReplace(dirRule, true)

	nameRule := policy.FileNameRule
	nameRule = dynamicReplace(nameRule, false)

	return path.Join(path.Clean(dirRule), nameRule)
}

func (m *RelocateTask) Progress(ctx context.Context) queue.Progresses {
	m.Lock()
	defer m.Unlock()
	if m.progress == nil {
		m.progress = make(queue.Progresses)
	}
	return m.progress
}

func (m *RelocateTask) Summarize(hasher hashid.Encoder) *queue.Summary {
	if m.state == nil {
		if err := json.Unmarshal([]byte(m.State()), &m.state); err != nil {
			return nil
		}
	}

	return &queue.Summary{
		Phase: string(m.state.Phase),
		Props: map[string]any{
			SummaryKeySrc:         m.state.SrcURI,
			SummaryKeySrcDstPolicyID: m.state.TargetPolicyID,
			SummaryKeyFailed:      m.state.Failed,
			SummaryKeyTotal:       m.state.Total,
		},
	}
}
