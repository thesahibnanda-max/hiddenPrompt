// Advisory client-side mirror of pkg/utils/validators.go's ValidatePassword.
// The server remains authoritative; this only powers the live checklist UI.

export interface PasswordRuleCheck {
  key: string;
  label: string;
  passed: boolean;
}

export function checkPasswordRules(password: string): PasswordRuleCheck[] {
  return [
    { key: "length", label: "8-32 characters", passed: password.length >= 8 && password.length <= 32 },
    { key: "upper", label: "One uppercase letter", passed: /[A-Z]/.test(password) },
    { key: "lower", label: "One lowercase letter", passed: /[a-z]/.test(password) },
    { key: "digit", label: "One digit", passed: /[0-9]/.test(password) },
    {
      key: "symbol",
      label: "One symbol (e.g. ! ? # $)",
      passed: /[!-/:-@[-`{-~]/.test(password),
    },
  ];
}

export function isPasswordValid(password: string): boolean {
  return checkPasswordRules(password).every((r) => r.passed);
}
