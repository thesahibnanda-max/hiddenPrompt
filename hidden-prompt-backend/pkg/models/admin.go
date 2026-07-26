package models

// AdminDeleteUserRequest is the body for POST /admin/user/delete. Auth is
// via the X-Admin-Key header (see pkg/handler/helper.go's
// requireAdminAuth), not a body field.
type AdminDeleteUserRequest struct {
	Email string `json:"email"`
}
