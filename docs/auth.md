  # /auth command

  Summary

  - Purpose: Switch the active authentication method and trigger sign-in.
  - Scope: Interactive CLI only; opens an auth dialog.

  Syntax

  - /auth
  - No subcommands or arguments; any args are ignored.

  Behavior

  - Opens the Auth dialog that lets the user choose an auth method and validates prerequisites.
  - On valid selection, the setting is saved and an authentication flow runs.
  - If auth fails or times out, the dialog reopens with an error message.

  Dialog Options

  - Options shown:
      - Login with Google (AuthType.LOGIN_WITH_GOOGLE), OpenAI or QWEN
      - Login with OpenAI
      - Login with Anthropc
      - Use API Keys if they are defined
  - Initial selection order:
      - settings.merged.selectedAuthType, else
      - GEMINI_DEFAULT_AUTH_TYPE (must match an AuthType), else
      - If GEMINI_API_KEY is set → “Use Gemini API Key”, else
      - “Login with Google”

  Validation

  - Performed by validateAuthMethod:
      - LOGIN_WITH_GOOGLE: always allowed.
      - CLOUD_SHELL: always allowed (option visible only under CLOUD_SHELL="true").
      - USE_GEMINI: requires GEMINI_API_KEY; error if missing.
      - USE_VERTEX_AI: requires either:
      - `GOOGLE_CLOUD_PROJECT` and `GOOGLE_CLOUD_LOCATION`, or
      - `GOOGLE_API_KEY` (Express mode).
  - Additional checks:
      - If GEMINI_DEFAULT_AUTH_TYPE is set to an invalid value → error with allowed values.
      - If GEMINI_API_KEY is present and default type is unset or USE_GEMINI → shows hint to select
  “Gemini API Key”.

  Authentication Flow

  - On selection:
      - Clears cached credentials.
      - Saves selectedAuthType in the config
  - After dialog closes:
      - config.refreshAuth(authType) runs.
      - UI shows “Waiting for auth... (Press ESC or CTRL+C to cancel)”.
      - Timeout: 180 seconds → sets error “Authentication timed out. Please try again.” and reopens
  the dialog.
  - Enter: Select highlighted option.
  - Escape:
      - If an error message is currently displayed → ignored (cannot close).
      - If no selectedAuthType is set → shows “You must select an auth method to proceed. Press
  Ctrl+C twice to exit.”
      - Otherwise closes the dialog without changing the auth method.
  - While authenticating: ESC or Ctrl+C cancels auth and reopens the dialog.

  Environment Variables

  - GEMINI_API_KEY: Enables and validates “Use Gemini API Key”.
  - GEMINI_DEFAULT_AUTH_TYPE: Preselects an auth method; must be one of AuthType values.
  - GOOGLE_CLOUD_PROJECT + GOOGLE_CLOUD_LOCATION: Required for Vertex AI (non-Express).
  - GOOGLE_API_KEY: Alternative for Vertex AI Express.
  - CLOUD_SHELL="true": Enables “Use Cloud Shell user credentials”.

  Side Effects

  - Persists selectedAuthType to user-level settings.
  - Clears cached credential file before switching.
  - May exit the process for Google login in headless environments.

  Errors & Messages

  - Missing keys/vars: Detailed guidance shown inline in the dialog.
  - Invalid default type: “Invalid value for GEMINI_DEFAULT_AUTH_TYPE: "". Valid values are: .”
  - Timeout: “Authentication timed out. Please try again.”
  - Failure: “Failed to login. Message: .”
  - Headless Google login: Prints a restart banner and exits.
