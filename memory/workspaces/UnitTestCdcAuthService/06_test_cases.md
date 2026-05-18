# 06_test_cases — UnitTestCdcAuthService

## config/config_test.go
| ID | Function | Case | Expected |
|---|---|---|---|
| C1 | validateConfig | port empty | err contains "server.port required" |
| C2 | validateConfig | db.host empty | err contains "db.host required" |
| C3 | validateConfig | db.database empty | err contains "db.database required" |
| C4 | validateConfig | db.username empty | err contains "db.username required" |
| C5 | validateConfig | jwt.secret empty | err contains "jwt.secret required" |
| C6 | validateConfig | mode=production + placeholder secret | err contains "placeholder in production" |
| C7 | validateConfig | mode=dev + placeholder secret | nil |
| C8 | validateConfig | all valid | nil |
| C9 | NewConfig | load from valid YAML | cfg loaded, no err |
| C10 | NewConfig | env override JWT_SECRET | cfg.JWT.Secret = env value |
| C11 | NewConfig | YAML missing + env-only valid | cfg loaded |
| C12 | NewConfig | invalid (no jwt.secret) | err |

## internal/model/user_test.go
| ID | Function | Case | Expected |
|---|---|---|---|
| M1 | (User).TableName | always | "cdc_auth_service.auth_users" |

## internal/service/auth_service_test.go
| ID | Function | Case | Expected |
|---|---|---|---|
| S1 | Login | user not found | err "invalid username or password" |
| S2 | Login | password mismatch | err "invalid username or password" |
| S3 | Login | happy path | TokenResponse non-nil, AccessToken parseable, claims correct |
| S4 | Register | username exists | err "username already taken" |
| S5 | Register | email exists | err "email already registered" |
| S6 | Register | role invalid | err "invalid role" |
| S7 | Register | role empty → default operator | user.Role=="operator" |
| S8 | Register | repo Create fails | err contains "create user" |
| S9 | Register | bcrypt happy | user.Password is bcrypt hash (CompareHashAndPassword(stored, plain) nil) |
| S10 | RefreshToken | malformed token | err "invalid refresh token" |
| S11 | RefreshToken | wrong signing method | err |
| S12 | RefreshToken | type != refresh | err "not a refresh token" |
| S13 | RefreshToken | user not found | err "user not found" |
| S14 | RefreshToken | happy | new TokenResponse |
| S15 | generateTokens (indirect via Login) | claims content | user_id, username, email, role, type, iat, exp present |

## internal/api/auth_handler_test.go
| ID | Function | Case | Expected |
|---|---|---|---|
| H1 | Login | body invalid JSON | 400 + error |
| H2 | Login | missing username | 400 + "username and password are required" |
| H3 | Login | service error | 401 |
| H4 | Login | happy | 200 + AccessToken in body |
| H5 | Register | invalid JSON | 400 |
| H6 | Register | missing email | 400 |
| H7 | Register | service error (duplicate) | 409 |
| H8 | Register | happy | 201 + "user registered" |
| H9 | RefreshToken | invalid JSON | 400 |
| H10 | RefreshToken | empty refresh_token | 400 |
| H11 | RefreshToken | service error | 401 |
| H12 | RefreshToken | happy | 200 |
| H13 | Health | always | 200 + {status:ok, service:cdc-auth} |
