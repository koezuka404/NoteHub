package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
)

func usecaseTestNow() time.Time {
	return time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
}

func newStubUserRepoFull(users map[uuid.UUID]*entity.User) *stubUserRepoFull {
	return &stubUserRepoFull{mockAccountUserRepo: mockAccountUserRepo{users: users}}
}

type stubAccessCheck struct {
	result *WorkspaceAccessResult
	err    error
}

func (s *stubAccessCheck) CheckWorkspaceAccess(_ context.Context, input CheckWorkspaceAccessInput) (*WorkspaceAccessResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.result == nil {
		return &WorkspaceAccessResult{}, nil
	}
	if input.RequiredRole != nil && s.result.Member != nil && s.result.Member.Role != *input.RequiredRole {
		if *input.RequiredRole == entity.WorkspaceRoleHost {
			return nil, ErrHostPermissionRequired
		}
		return nil, ErrWorkspacePermissionDenied
	}
	return s.result, nil
}

type stubDocumentRepo struct {
	byID              map[uuid.UUID]*entity.Document
	byWorkspace       map[uuid.UUID][]entity.Document
	createErr         error
	updateErr         error
	findErr           error
	findForUpdateErr  error
	findForUpdateMiss bool
	findForUpdateDoc  *entity.Document
	listErr           error
}

func (s *stubDocumentRepo) Create(_ context.Context, doc *entity.Document) error {
	if s.createErr != nil {
		return s.createErr
	}
	if s.byID == nil {
		s.byID = map[uuid.UUID]*entity.Document{}
	}
	copy := *doc
	s.byID[doc.ID] = &copy
	return nil
}

func (s *stubDocumentRepo) FindByID(_ context.Context, id uuid.UUID) (*entity.Document, bool, error) {
	if s.findErr != nil {
		return nil, false, s.findErr
	}
	doc, ok := s.byID[id]
	if !ok {
		return nil, false, nil
	}
	copy := *doc
	return &copy, true, nil
}

func (s *stubDocumentRepo) FindByIDForUpdate(_ context.Context, id uuid.UUID) (*entity.Document, bool, error) {
	if s.findForUpdateErr != nil {
		return nil, false, s.findForUpdateErr
	}
	if s.findForUpdateMiss {
		return nil, false, nil
	}
	if s.findForUpdateDoc != nil && s.findForUpdateDoc.ID == id {
		copy := *s.findForUpdateDoc
		return &copy, true, nil
	}
	return s.FindByID(context.Background(), id)
}

func (s *stubDocumentRepo) FindByWorkspaceID(_ context.Context, workspaceID uuid.UUID) ([]entity.Document, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.byWorkspace[workspaceID], nil
}

func (s *stubDocumentRepo) Update(_ context.Context, doc *entity.Document) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	if s.byID == nil {
		s.byID = map[uuid.UUID]*entity.Document{}
	}
	copy := *doc
	s.byID[doc.ID] = &copy
	return nil
}

type stubVersionRepo struct {
	byID map[uuid.UUID]*entity.DocumentVersion
	err  error
}

func (s *stubVersionRepo) Create(_ context.Context, version *entity.DocumentVersion) error {
	if s.err != nil {
		return s.err
	}
	if s.byID == nil {
		s.byID = map[uuid.UUID]*entity.DocumentVersion{}
	}
	copy := *version
	s.byID[version.ID] = &copy
	return nil
}

func (s *stubVersionRepo) FindByID(_ context.Context, id uuid.UUID) (*entity.DocumentVersion, bool, error) {
	if s.err != nil {
		return nil, false, s.err
	}
	v, ok := s.byID[id]
	if !ok {
		return nil, false, nil
	}
	copy := *v
	return &copy, true, nil
}

func (s *stubVersionRepo) FindByDocumentID(_ context.Context, documentID uuid.UUID) ([]entity.DocumentVersion, error) {
	if s.err != nil {
		return nil, s.err
	}
	var out []entity.DocumentVersion
	for _, v := range s.byID {
		if v.DocumentID == documentID {
			out = append(out, *v)
		}
	}
	return out, nil
}

type stubDocumentCache struct {
	content   map[uuid.UUID]DocumentContentState
	dirty     map[uuid.UUID]bool
	revisions map[uuid.UUID]uint64
	idleIDs   []uuid.UUID
	dirtyIDs  []uuid.UUID
	getErr    error
	setErr    error
	isDirty   bool
	hasState  bool
}

func (s *stubDocumentCache) GetContentState(_ context.Context, documentID uuid.UUID) (DocumentContentState, bool, error) {
	if s.getErr != nil {
		return DocumentContentState{}, false, s.getErr
	}
	if s.content != nil {
		if state, ok := s.content[documentID]; ok {
			return state, true, nil
		}
	}
	return DocumentContentState{}, s.hasState, nil
}

func (s *stubDocumentCache) SetContent(_ context.Context, documentID uuid.UUID, content string, updatedBy uuid.UUID, updatedAt time.Time) error {
	if s.setErr != nil {
		return s.setErr
	}
	if s.content == nil {
		s.content = map[uuid.UUID]DocumentContentState{}
	}
	s.content[documentID] = DocumentContentState{Content: content, UpdatedBy: updatedBy, UpdatedAt: updatedAt}
	if s.dirty == nil {
		s.dirty = map[uuid.UUID]bool{}
	}
	s.dirty[documentID] = true
	return nil
}

func (s *stubDocumentCache) IsDirty(_ context.Context, documentID uuid.UUID) (bool, error) {
	if s.getErr != nil {
		return false, s.getErr
	}
	if s.dirty != nil {
		return s.dirty[documentID], nil
	}
	return s.isDirty, nil
}

func (s *stubDocumentCache) MarkClean(_ context.Context, documentID uuid.UUID) error {
	if s.dirty != nil {
		delete(s.dirty, documentID)
	}
	return s.setErr
}

func (s *stubDocumentCache) GetRevision(_ context.Context, documentID uuid.UUID) (uint64, error) {
	if s.getErr != nil {
		return 0, s.getErr
	}
	if s.revisions != nil {
		return s.revisions[documentID], nil
	}
	return 1, nil
}

func (s *stubDocumentCache) ListDirtyDocumentIDs(context.Context) ([]uuid.UUID, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.dirtyIDs, nil
}

func (s *stubDocumentCache) ListIdleDirtyDocumentIDs(context.Context, time.Duration) ([]uuid.UUID, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.idleIDs, nil
}

func (s *stubDocumentCache) Clear(context.Context, uuid.UUID) error {
	return s.setErr
}

type stubAutoSaveLock struct {
	locked   bool
	tryErr   error
	unlockErr error
}

func (s *stubAutoSaveLock) TryLock(context.Context, string, time.Duration) (bool, error) {
	if s.tryErr != nil {
		return false, s.tryErr
	}
	return s.locked, nil
}

func (s *stubAutoSaveLock) Unlock(context.Context, string) error {
	return s.unlockErr
}

type stubDocumentNotifier struct {
	err error
}

func (s *stubDocumentNotifier) NotifyDocumentCreated(uuid.UUID, uuid.UUID, string, string, string) error {
	return s.err
}
func (s *stubDocumentNotifier) NotifyDocumentTitleUpdated(uuid.UUID, uuid.UUID, string, string, string) error {
	return s.err
}
func (s *stubDocumentNotifier) NotifyDocumentRestored(uuid.UUID, string, uuid.UUID) error { return s.err }
func (s *stubDocumentNotifier) NotifyDocumentDeleted(uuid.UUID, uuid.UUID, string) error  { return s.err }
func (s *stubDocumentNotifier) NotifyDocumentListDeleted(uuid.UUID, uuid.UUID, uuid.UUID, string) error {
	return s.err
}
func (s *stubDocumentNotifier) NotifyWorkspaceDeleted(uuid.UUID, uuid.UUID, string) error { return s.err }

type stubDocumentFlush struct {
	err error
}

func (s *stubDocumentFlush) FlushDocument(context.Context, uuid.UUID) error { return s.err }
func (s *stubDocumentFlush) FlushAllDirty(context.Context) error            { return s.err }
func (s *stubDocumentFlush) FlushWorkspaceDocuments(context.Context, uuid.UUID) error {
	return s.err
}

type stubWorkspaceRepoFull struct {
	mockWorkspaceRepo
	byID              map[uuid.UUID]*entity.Workspace
	createErr         error
	updateErr         error
	findErr           error
	findForUpdateErr  error
	findForUpdateMiss bool
	findForUpdateDoc  *entity.Workspace
	listErr           error
}

func (s *stubWorkspaceRepoFull) Create(_ context.Context, ws *entity.Workspace) error {
	if s.createErr != nil {
		return s.createErr
	}
	if s.byID == nil {
		s.byID = map[uuid.UUID]*entity.Workspace{}
	}
	copy := *ws
	s.byID[ws.ID] = &copy
	return nil
}

func (s *stubWorkspaceRepoFull) FindByID(_ context.Context, id uuid.UUID) (*entity.Workspace, bool, error) {
	if s.findErr != nil {
		return nil, false, s.findErr
	}
	ws, ok := s.byID[id]
	if !ok {
		return nil, false, nil
	}
	copy := *ws
	return &copy, true, nil
}

func (s *stubWorkspaceRepoFull) FindByIDForUpdate(_ context.Context, id uuid.UUID) (*entity.Workspace, bool, error) {
	if s.findForUpdateErr != nil {
		return nil, false, s.findForUpdateErr
	}
	if s.findForUpdateMiss {
		return nil, false, nil
	}
	if s.findForUpdateDoc != nil {
		copy := *s.findForUpdateDoc
		return &copy, true, nil
	}
	return s.FindByID(context.Background(), id)
}

func (s *stubWorkspaceRepoFull) FindByUserID(_ context.Context, userID uuid.UUID) ([]entity.Workspace, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	var out []entity.Workspace
	for _, ws := range s.byID {
		if ws.HostID == userID {
			out = append(out, *ws)
		}
	}
	return out, nil
}

func (s *stubWorkspaceRepoFull) Update(_ context.Context, ws *entity.Workspace) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	if s.byID == nil {
		s.byID = map[uuid.UUID]*entity.Workspace{}
	}
	copy := *ws
	s.byID[ws.ID] = &copy
	return nil
}

type stubUserRepoFull struct {
	mockAccountUserRepo
	findByEmailUser  *entity.User
	findByEmailFound bool
	findByEmailErr   error
	findByIDErr      error
	incrementErr     error
	updateErr        error
	createErr        error
}

func (s *stubUserRepoFull) FindByEmail(_ context.Context, email string) (*entity.User, bool, error) {
	if s.findByEmailErr != nil {
		return nil, false, s.findByEmailErr
	}
	if !s.findByEmailFound {
		return nil, false, nil
	}
	if s.findByEmailUser != nil && s.findByEmailUser.Email != email {
		return nil, false, nil
	}
	copy := *s.findByEmailUser
	return &copy, true, nil
}

func (s *stubUserRepoFull) FindByID(_ context.Context, id uuid.UUID) (*entity.User, bool, error) {
	if s.findByIDErr != nil {
		return nil, false, s.findByIDErr
	}
	user, ok := s.users[id]
	if !ok {
		return nil, false, nil
	}
	copy := *user
	return &copy, true, nil
}

func (s *stubUserRepoFull) IncrementAuthVersion(context.Context, uuid.UUID, time.Time) error {
	return s.incrementErr
}

func (s *stubUserRepoFull) Update(_ context.Context, user *entity.User) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	return s.mockAccountUserRepo.Update(context.Background(), user)
}

func (s *stubUserRepoFull) Create(_ context.Context, user *entity.User) error {
	if s.createErr != nil {
		return s.createErr
	}
	if s.users == nil {
		s.users = map[uuid.UUID]*entity.User{}
	}
	copy := *user
	s.users[user.ID] = &copy
	return nil
}

func (s *stubUserRepoFull) ExistsByEmail(_ context.Context, email string) (bool, error) {
	for _, u := range s.users {
		if u.Email == email {
			return true, nil
		}
	}
	return false, nil
}

type stubMemberRepoFull struct {
	mockMemberRepo
	createErr error
	deleteErr error
	exists    bool
	list      []entity.WorkspaceMember
	listErr   error
}

func (s *stubMemberRepoFull) Create(_ context.Context, member *entity.WorkspaceMember) error {
	if s.createErr != nil {
		return s.createErr
	}
	return nil
}

func (s *stubMemberRepoFull) Delete(_ context.Context, workspaceID, userID uuid.UUID) error {
	return s.deleteErr
}

func (s *stubMemberRepoFull) Exists(_ context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return s.exists, nil
}

func (s *stubMemberRepoFull) FindByWorkspaceID(_ context.Context, workspaceID uuid.UUID) ([]entity.WorkspaceMember, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.list, nil
}

type stubCleanupRedis struct {
	removed int
	err     error
}

func (s *stubCleanupRedis) CleanupEphemeralKeys(context.Context) (int, error) {
	return s.removed, s.err
}

type stubRefreshRepoFull struct {
	mockRefreshTokenRepo
	byHash      map[string]*entity.RefreshToken
	updateErr   error
	findErr     error
	revokeFamilyErr error
	incrementErr    error
	markErr         error
	deleteErr       error
}

func (s *stubRefreshRepoFull) FindByHashForUpdate(_ context.Context, hash string) (*entity.RefreshToken, bool, error) {
	if s.findErr != nil {
		return nil, false, s.findErr
	}
	token, ok := s.byHash[hash]
	if !ok {
		return nil, false, nil
	}
	copy := *token
	return &copy, true, nil
}

func (s *stubRefreshRepoFull) Update(_ context.Context, token *entity.RefreshToken) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	if s.byHash == nil {
		s.byHash = map[string]*entity.RefreshToken{}
	}
	copy := *token
	s.byHash[token.TokenHash] = &copy
	return nil
}

func (s *stubRefreshRepoFull) RevokeFamily(context.Context, uuid.UUID, time.Time) error {
	return s.revokeFamilyErr
}

func (s *stubRefreshRepoFull) RevokeAllByUserID(context.Context, uuid.UUID, time.Time) error {
	return s.revokeFamilyErr
}

func (s *stubRefreshRepoFull) MarkExpiredBefore(context.Context, time.Time) (int64, error) {
	if s.markErr != nil {
		return 0, s.markErr
	}
	return 1, nil
}

func (s *stubRefreshRepoFull) DeleteStaleBefore(context.Context, time.Time) (int64, error) {
	if s.deleteErr != nil {
		return 0, s.deleteErr
	}
	return 2, nil
}

type stubWebSocketSessions struct {
	added     bool
	count     int
	addErr    error
	removeErr error
	remaining int
}

func (s *stubWebSocketSessions) TryAddConnection(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, int, time.Duration) (int, bool, error) {
	if s.addErr != nil {
		return 0, false, s.addErr
	}
	return s.count, s.added, nil
}

func (s *stubWebSocketSessions) RemoveConnection(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (int, error) {
	if s.removeErr != nil {
		return 0, s.removeErr
	}
	return s.remaining, nil
}

type stubDocumentEditors struct {
	joined    bool
	left      bool
	editors   []DocumentEditorInfo
	addErr    error
	removeErr error
	listErr   error
	refreshErr error
}

func (s *stubDocumentEditors) List(context.Context, uuid.UUID) ([]DocumentEditorInfo, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.editors, nil
}

func (s *stubDocumentEditors) Add(context.Context, uuid.UUID, DocumentEditorInfo) (bool, []DocumentEditorInfo, error) {
	if s.addErr != nil {
		return false, nil, s.addErr
	}
	return s.joined, s.editors, nil
}

func (s *stubDocumentEditors) Remove(context.Context, uuid.UUID, uuid.UUID) (bool, []DocumentEditorInfo, error) {
	if s.removeErr != nil {
		return false, nil, s.removeErr
	}
	return s.left, s.editors, nil
}

func (s *stubDocumentEditors) RefreshTTL(context.Context, uuid.UUID) error {
	return s.refreshErr
}

type stubMemberNotifier struct {
	err error
}

func (s *stubMemberNotifier) NotifyMemberRemoved(uuid.UUID, uuid.UUID, uuid.UUID, string) error {
	return s.err
}

type errAuditLogRepo struct{ err error }

func (e errAuditLogRepo) Create(context.Context, *entity.AuditLog) error { return e.err }

type errTransactionManager struct{ err error }

func (e errTransactionManager) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	if e.err != nil {
		return e.err
	}
	return fn(ctx)
}

type noopSuccessTransactionManager struct{}

func (noopSuccessTransactionManager) WithinTransaction(context.Context, func(context.Context) error) error {
	return nil
}

type failingAuthService struct {
	mockAuthService
	hashPasswordErr       error
	generateAccessErr     error
	generateRefreshErr    error
	generateCSRFErr       error
	revokeAccessErr       error
}

func (f *failingAuthService) HashPassword(string) (string, error) {
	if f.hashPasswordErr != nil {
		return "", f.hashPasswordErr
	}
	return f.mockAuthService.HashPassword("")
}
func (f *failingAuthService) GenerateAccessToken(uuid.UUID, uint, time.Time) (string, time.Time, error) {
	if f.generateAccessErr != nil {
		return "", time.Time{}, f.generateAccessErr
	}
	return f.mockAuthService.GenerateAccessToken(uuid.UUID{}, 0, time.Time{})
}
func (f *failingAuthService) GenerateRefreshToken() (string, error) {
	if f.generateRefreshErr != nil {
		return "", f.generateRefreshErr
	}
	return f.mockAuthService.GenerateRefreshToken()
}
func (f *failingAuthService) GenerateCSRFToken() (string, error) {
	if f.generateCSRFErr != nil {
		return "", f.generateCSRFErr
	}
	return f.mockAuthService.GenerateCSRFToken()
}
func (f *failingAuthService) RevokeAccessToken(ctx context.Context, jti uuid.UUID, ttl time.Duration) error {
	if f.revokeAccessErr != nil {
		return f.revokeAccessErr
	}
	return f.mockAuthService.RevokeAccessToken(ctx, jti, ttl)
}
