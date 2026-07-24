package usecase

import "github.com/koezuka404/notehub/entity"

func checkWorkspaceHostStatus(host *entity.User) error {
	if host.CanAuthenticate() {
		return nil
	}
	if host.IsSuspended() {
		return ErrWorkspaceHostSuspended
	}
	if host.IsDeleted() {
		return ErrWorkspaceHostDeleted
	}
	return ErrWorkspaceHostSuspended
}
