package customErrors

type UserAlreadyExists struct {
	Err error
}

type EMailIsInvalid struct {
	Err error
}

type PasswordIsInvalid struct {
	Err error
}

type InvalidCredentials struct {
	Err error
}

type UserNotFound struct {
	Err error
}

type OTPInvalidOrExpired struct {
	Err error
}

type UserAlreadyVerified struct {
	Err error
}

type UserIsNotVerified struct {
	Err error
}

type PuzzleNotFound struct {
	Err error
}

type PuzzleIDIsInvalid struct {
	Err error
}

type PuzzleAlreadyWon struct {
	Err error
}

type UserPromptIsInvalid struct {
	Err error
}

type MusicDetailsRequired struct {
	Err error
}

func (u *UserAlreadyExists) Error() string {
	if u == nil || u.Err == nil {
		return ""
	}
	return u.Err.Error()
}

func (e *EMailIsInvalid) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (p *PasswordIsInvalid) Error() string {
	if p == nil || p.Err == nil {
		return ""
	}
	return p.Err.Error()
}

func (i *InvalidCredentials) Error() string {
	if i == nil || i.Err == nil {
		return ""
	}
	return i.Err.Error()
}

func (u *UserNotFound) Error() string {
	if u == nil || u.Err == nil {
		return ""
	}
	return u.Err.Error()
}

func (o *OTPInvalidOrExpired) Error() string {
	if o == nil || o.Err == nil {
		return ""
	}
	return o.Err.Error()
}

func (u *UserAlreadyVerified) Error() string {
	if u == nil || u.Err == nil {
		return ""
	}
	return u.Err.Error()
}

func (u *UserIsNotVerified) Error() string {
	if u == nil || u.Err == nil {
		return ""
	}
	return u.Err.Error()
}

func (p *PuzzleNotFound) Error() string {
	if p == nil || p.Err == nil {
		return ""
	}
	return p.Err.Error()
}

func (p *PuzzleIDIsInvalid) Error() string {
	if p == nil || p.Err == nil {
		return ""
	}
	return p.Err.Error()
}

func (p *PuzzleAlreadyWon) Error() string {
	if p == nil || p.Err == nil {
		return ""
	}
	return p.Err.Error()
}

func (u *UserPromptIsInvalid) Error() string {
	if u == nil || u.Err == nil {
		return ""
	}
	return u.Err.Error()
}

func (m *MusicDetailsRequired) Error() string {
	if m == nil || m.Err == nil {
		return ""
	}
	return m.Err.Error()
}
