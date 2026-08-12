package controller

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/entity"
	infrcrypto "github.com/koezuka404/notehub/usecase/crypto"
	"github.com/koezuka404/notehub/usecase"
)

type mockAuthUsecase struct {
	registerOut *usecase.RegisterOutput
	registerErr error
	loginOut    *usecase.LoginOutput
	loginErr    error
	refreshOut  *usecase.RefreshOutput
	refreshErr  error
	logoutOut   *usecase.LogoutOutput
	logoutErr   error
	meOut       *usecase.GetCurrentUserOutput
	meErr       error
	issueCSRFOut string
	issueCSRFErr error
}

func (m *mockAuthUsecase) Register(context.Context, usecase.RegisterInput) (*usecase.RegisterOutput, error) {
	return m.registerOut, m.registerErr
}
func (m *mockAuthUsecase) Login(context.Context, usecase.LoginInput) (*usecase.LoginOutput, error) {
	return m.loginOut, m.loginErr
}
func (m *mockAuthUsecase) Refresh(context.Context, usecase.RefreshInput) (*usecase.RefreshOutput, error) {
	return m.refreshOut, m.refreshErr
}
func (m *mockAuthUsecase) Logout(context.Context, usecase.LogoutInput) (*usecase.LogoutOutput, error) {
	return m.logoutOut, m.logoutErr
}
func (m *mockAuthUsecase) GetCurrentUser(context.Context, usecase.GetCurrentUserInput) (*usecase.GetCurrentUserOutput, error) {
	return m.meOut, m.meErr
}
func (m *mockAuthUsecase) IssueCSRFToken(context.Context) (string, error) {
	return m.issueCSRFOut, m.issueCSRFErr
}

type mockAccountUsecase struct {
	suspendOut     *usecase.SuspendAccountOutput
	suspendErr     error
	reactivateOut  *usecase.ReactivateAccountOutput
	reactivateErr  error
	deleteOut      *usecase.DeleteAccountOutput
	deleteErr      error
}

func (m *mockAccountUsecase) SuspendAccount(context.Context, usecase.SuspendAccountInput) (*usecase.SuspendAccountOutput, error) {
	return m.suspendOut, m.suspendErr
}
func (m *mockAccountUsecase) ReactivateAccount(context.Context, usecase.ReactivateAccountInput) (*usecase.ReactivateAccountOutput, error) {
	return m.reactivateOut, m.reactivateErr
}
func (m *mockAccountUsecase) DeleteAccount(context.Context, usecase.DeleteAccountInput) (*usecase.DeleteAccountOutput, error) {
	return m.deleteOut, m.deleteErr
}

type mockWorkspaceUsecase struct {
	createOut *usecase.CreateWorkspaceOutput
	createErr error
	listOut   []usecase.WorkspaceListItem
	listErr   error
	getOut    *usecase.GetWorkspaceOutput
	getErr    error
	updateOut *usecase.UpdateWorkspaceOutput
	updateErr error
	deleteOut *usecase.DeleteWorkspaceOutput
	deleteErr error
}

func (m *mockWorkspaceUsecase) CreateWorkspace(context.Context, usecase.CreateWorkspaceInput) (*usecase.CreateWorkspaceOutput, error) {
	return m.createOut, m.createErr
}
func (m *mockWorkspaceUsecase) ListWorkspaces(context.Context, usecase.ListWorkspacesInput) ([]usecase.WorkspaceListItem, error) {
	return m.listOut, m.listErr
}
func (m *mockWorkspaceUsecase) GetWorkspace(context.Context, usecase.GetWorkspaceInput) (*usecase.GetWorkspaceOutput, error) {
	return m.getOut, m.getErr
}
func (m *mockWorkspaceUsecase) UpdateWorkspace(context.Context, usecase.UpdateWorkspaceInput) (*usecase.UpdateWorkspaceOutput, error) {
	return m.updateOut, m.updateErr
}
func (m *mockWorkspaceUsecase) DeleteWorkspace(context.Context, usecase.DeleteWorkspaceInput) (*usecase.DeleteWorkspaceOutput, error) {
	return m.deleteOut, m.deleteErr
}

type mockDocumentUsecase struct {
	createOut *usecase.CreateDocumentOutput
	createErr error
	listOut   []usecase.DocumentListItem
	listErr   error
	getOut    *usecase.GetDocumentOutput
	getErr    error
	updateOut *usecase.UpdateDocumentOutput
	updateErr error
	deleteOut *usecase.DeleteDocumentOutput
	deleteErr error
}

func (m *mockDocumentUsecase) CreateDocument(context.Context, usecase.CreateDocumentInput) (*usecase.CreateDocumentOutput, error) {
	return m.createOut, m.createErr
}
func (m *mockDocumentUsecase) ListDocuments(context.Context, usecase.ListDocumentsInput) ([]usecase.DocumentListItem, error) {
	return m.listOut, m.listErr
}
func (m *mockDocumentUsecase) GetDocument(context.Context, usecase.GetDocumentInput) (*usecase.GetDocumentOutput, error) {
	return m.getOut, m.getErr
}
func (m *mockDocumentUsecase) UpdateDocument(context.Context, usecase.UpdateDocumentInput) (*usecase.UpdateDocumentOutput, error) {
	return m.updateOut, m.updateErr
}
func (m *mockDocumentUsecase) DeleteDocument(context.Context, usecase.DeleteDocumentInput) (*usecase.DeleteDocumentOutput, error) {
	return m.deleteOut, m.deleteErr
}

type mockMemberUsecase struct {
	searchOut *usecase.SearchUserOutput
	searchErr error
	listOut   []usecase.MemberListItem
	listErr   error
	addOut    *usecase.AddMemberOutput
	addErr    error
	removeErr error
}

func (m *mockMemberUsecase) SearchUser(context.Context, usecase.SearchUserInput) (*usecase.SearchUserOutput, error) {
	return m.searchOut, m.searchErr
}
func (m *mockMemberUsecase) ListMembers(context.Context, usecase.ListMembersInput) ([]usecase.MemberListItem, error) {
	return m.listOut, m.listErr
}
func (m *mockMemberUsecase) AddMember(context.Context, usecase.AddMemberInput) (*usecase.AddMemberOutput, error) {
	return m.addOut, m.addErr
}
func (m *mockMemberUsecase) RemoveMember(context.Context, usecase.RemoveMemberInput) error {
	return m.removeErr
}

type mockVersionUsecase struct {
	listOut    []usecase.VersionListItem
	listErr    error
	getOut     *usecase.GetVersionOutput
	getErr     error
	saveOut    *usecase.SaveManualVersionOutput
	saveErr    error
	restoreOut *usecase.RestoreVersionOutput
	restoreErr error
}

func (m *mockVersionUsecase) ListVersions(context.Context, usecase.ListVersionsInput) ([]usecase.VersionListItem, error) {
	return m.listOut, m.listErr
}
func (m *mockVersionUsecase) GetVersion(context.Context, usecase.GetVersionInput) (*usecase.GetVersionOutput, error) {
	return m.getOut, m.getErr
}
func (m *mockVersionUsecase) CreateVersion(context.Context, usecase.CreateVersionInput) error {
	return nil
}
func (m *mockVersionUsecase) SaveManualVersion(context.Context, usecase.SaveManualVersionInput) (*usecase.SaveManualVersionOutput, error) {
	return m.saveOut, m.saveErr
}
func (m *mockVersionUsecase) RestoreVersion(context.Context, usecase.RestoreVersionInput) (*usecase.RestoreVersionOutput, error) {
	return m.restoreOut, m.restoreErr
}

type mockWSUsecase struct {
	prepareOut      *usecase.PrepareWebSocketConnectionOutput
	prepareErr      error
	prepareWSErr    error
	registerOut     *usecase.RegisterWebSocketConnectionOutput
	registerErr     error
	unregisterOut   *usecase.UnregisterWebSocketConnectionOutput
	unregisterErr   error
	applyEditOut    *usecase.ApplyDocumentEditOutput
	applyEditErr    error
	onUnregister    func()
}

func (m *mockWSUsecase) PrepareConnection(context.Context, usecase.PrepareWebSocketConnectionInput) (*usecase.PrepareWebSocketConnectionOutput, error) {
	return m.prepareOut, m.prepareErr
}
func (m *mockWSUsecase) PrepareWorkspaceConnection(context.Context, usecase.PrepareWorkspaceConnectionInput) error {
	return m.prepareWSErr
}
func (m *mockWSUsecase) RegisterConnection(context.Context, usecase.RegisterWebSocketConnectionInput) (*usecase.RegisterWebSocketConnectionOutput, error) {
	return m.registerOut, m.registerErr
}
func (m *mockWSUsecase) UnregisterConnection(ctx context.Context, input usecase.UnregisterWebSocketConnectionInput) (*usecase.UnregisterWebSocketConnectionOutput, error) {
	if m.onUnregister != nil {
		m.onUnregister()
	}
	return m.unregisterOut, m.unregisterErr
}
func (m *mockWSUsecase) ApplyDocumentEdit(context.Context, usecase.ApplyDocumentEditInput) (*usecase.ApplyDocumentEditOutput, error) {
	return m.applyEditOut, m.applyEditErr
}

type mockTokenValidator struct {
	claims infrcrypto.AccessTokenClaims
	err    error
}

func (m *mockTokenValidator) ValidateAccessToken(string, time.Time) (infrcrypto.AccessTokenClaims, error) {
	return m.claims, m.err
}

type mockUserFinder struct {
	user  *entity.User
	found bool
	err   error
}

func (m *mockUserFinder) FindByID(context.Context, uuid.UUID) (*entity.User, bool, error) {
	return m.user, m.found, m.err
}

type mockRevocationChecker struct {
	revoked bool
	err     error
}

func (m *mockRevocationChecker) IsRevoked(context.Context, uuid.UUID) (bool, error) {
	return m.revoked, m.err
}
