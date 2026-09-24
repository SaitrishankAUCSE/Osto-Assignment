package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"authcli/internal/auth"
	"authcli/internal/security"
	"authcli/internal/store"

	"github.com/chzyer/readline"
)

type App struct {
	authSvc *auth.Service
	rl      *readline.Instance
	token   string
	user    *store.User
}

func NewApp(authSvc *auth.Service) *App {
	return &App{
		authSvc: authSvc,
	}
}

func (a *App) Run() error {
	completer := readline.NewPrefixCompleter(
		readline.PcItem("register"),
		readline.PcItem("login"),
		readline.PcItem("help"),
		readline.PcItem("exit"),
	)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "auth> ",
		HistoryFile:     filepath.Join(os.TempDir(), ".authcli_history"),
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return err
	}
	defer rl.Close()
	a.rl = rl

	fmt.Println("Welcome to AuthCLI. Type 'help' to see available commands.")

	for {
		line, err := a.rl.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		a.handleCommand(line)
	}

	return nil
}

func (a *App) updateCompleter() {
	var completer readline.AutoCompleter
	if a.token == "" {
		completer = readline.NewPrefixCompleter(
			readline.PcItem("register"),
			readline.PcItem("login"),
			readline.PcItem("help"),
			readline.PcItem("exit"),
		)
	} else {
		completer = readline.NewPrefixCompleter(
			readline.PcItem("whoami"),
			readline.PcItem("enable-2fa"),
			readline.PcItem("disable-2fa"),
			readline.PcItem("logout"),
			readline.PcItem("help"),
			readline.PcItem("exit"),
		)
	}
	a.rl.Config.AutoComplete = completer
}

func (a *App) handleCommand(cmd string) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}

	action := parts[0]

	// Require active session check for protected commands
	if a.token != "" && action != "exit" && action != "logout" && action != "help" {
		sess, u, err := a.authSvc.ValidateSession(a.token)
		if err != nil {
			fmt.Println("Session invalid or expired. Please login again.")
			a.token = ""
			a.user = nil
			a.updateCompleter()
			return
		}
		a.user = u
		_ = sess
	}

	if a.token == "" {
		switch action {
		case "register":
			a.cmdRegister()
		case "login":
			a.cmdLogin()
		case "help":
			a.printUnauthHelp()
		case "exit":
			fmt.Println("Goodbye!")
			os.Exit(0)
		default:
			fmt.Println("Unknown command or you must be logged in. Type 'help'.")
		}
	} else {
		switch action {
		case "whoami":
			a.cmdWhoami()
		case "enable-2fa":
			a.cmdEnable2FA()
		case "disable-2fa":
			a.cmdDisable2FA()
		case "logout":
			a.cmdLogout()
		case "help":
			a.printAuthHelp()
		case "exit":
			fmt.Println("Goodbye!")
			os.Exit(0)
		default:
			fmt.Println("Unknown command. Type 'help'.")
		}
	}
}

func (a *App) printUnauthHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  register - create a new user")
	fmt.Println("  login    - login with username/password")
	fmt.Println("  help     - show available commands")
	fmt.Println("  exit     - quit program")
}

func (a *App) printAuthHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  whoami      - show current user details")
	fmt.Println("  enable-2fa  - enable TOTP-based MFA")
	fmt.Println("  disable-2fa - disable MFA")
	fmt.Println("  logout      - end session")
	fmt.Println("  help        - show available commands")
	fmt.Println("  exit        - quit program")
}

func (a *App) promptPassword(prompt string) (string, error) {
	bytepw, err := a.rl.ReadPassword(prompt)
	if err != nil {
		return "", err
	}
	return string(bytepw), nil
}

func (a *App) promptInput(prompt string) (string, error) {
	a.rl.SetPrompt(prompt)
	line, err := a.rl.Readline()
	a.rl.SetPrompt("auth> ")
	return strings.TrimSpace(line), err
}

func (a *App) cmdRegister() {
	username, _ := a.promptInput("Username: ")
	if username == "" {
		fmt.Println("Registration aborted.")
		return
	}
	password, _ := a.promptPassword("Password: ")
	if password == "" {
		fmt.Println("Registration aborted.")
		return
	}

	err := a.authSvc.Register(username, password)
	if err != nil {
		fmt.Printf("Registration failed: %v\n", err)
		return
	}
	fmt.Println("Registration successful! You can now login.")
}

func (a *App) cmdLogin() {
	username, _ := a.promptInput("Username: ")
	if username == "" {
		fmt.Println("Login aborted.")
		return
	}
	password, _ := a.promptPassword("Password: ")

	sess, u, token, err := a.authSvc.Login(username, password)
	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		return
	}

	// MFA flow
	if u.MFAEnabled {
		code, _ := a.promptInput("2FA Code: ")
		sess, u, token, err = a.authSvc.ValidateTOTPLogin(u, code)
		if err != nil {
			fmt.Printf("MFA failed: %v\n", err)
			return
		}
	}

	a.token = token
	a.user = u
	a.updateCompleter()

	fmt.Println("\nLogin successful!")
	a.printUserDetails(sess, u)
}

func (a *App) cmdLogout() {
	if a.token != "" {
		_ = a.authSvc.Logout(a.token)
	}
	a.token = ""
	a.user = nil
	a.updateCompleter()
	fmt.Println("Logged out successfully.")
}

func (a *App) cmdWhoami() {
	sess, u, _ := a.authSvc.ValidateSession(a.token)
	if u != nil {
		a.user = u
	}
	a.printUserDetails(sess, a.user)
}

func (a *App) cmdEnable2FA() {
	if a.user.MFAEnabled {
		fmt.Println("2FA is already enabled.")
		return
	}

	key, err := security.GenerateTOTPSecret(a.user.Username)
	if err != nil {
		fmt.Printf("Error generating 2FA secret: %v\n", err)
		return
	}

	fmt.Println("Scan this QR code with Google Authenticator or similar app:")
	_ = security.PrintQR(key.URL())
	fmt.Printf("\nOr manually enter this secret: %s\n\n", key.Secret())

	code, _ := a.promptInput("Enter code to confirm: ")
	if security.ValidateTOTP(code, key.Secret()) {
		err := a.authSvc.EnableMFA(a.user.ID, key.Secret())
		if err != nil {
			fmt.Printf("Failed to enable 2FA: %v\n", err)
		} else {
			a.user.MFAEnabled = true
			fmt.Println("2FA enabled successfully!")
		}
	} else {
		fmt.Println("Invalid code. 2FA not enabled.")
	}
}

func (a *App) cmdDisable2FA() {
	if !a.user.MFAEnabled {
		fmt.Println("2FA is not enabled.")
		return
	}

	err := a.authSvc.DisableMFA(a.user.ID)
	if err != nil {
		fmt.Printf("Failed to disable 2FA: %v\n", err)
		return
	}
	a.user.MFAEnabled = false
	fmt.Println("2FA disabled successfully.")
}

func (a *App) printUserDetails(sess *store.Session, u *store.User) {
	fmt.Printf("Username: %s\n", u.Username)
	fmt.Printf("Registration date: %s\n", u.CreatedAt.Format("2006-01-02 15:04:05"))
	
	mfaStatus := "Disabled"
	if u.MFAEnabled {
		mfaStatus = "Enabled"
	}
	fmt.Printf("MFA: %s\n", mfaStatus)

	if sess != nil {
		fmt.Printf("Session expiration: %s\n", sess.ExpiresAt.Format("2006-01-02 15:04:05"))
	}
	
	if u.LastLoginAt != nil {
		fmt.Printf("Last login: %s\n", u.LastLoginAt.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("Last login: Never\n")
	}
}
