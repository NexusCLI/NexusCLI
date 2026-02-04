# auth.go Documentation

## Package utils

This package provides functionalities for user authentication and session management in the Zephyrus CLI application.

### Constants

- **configPath**: The path to the configuration file (`zephyrus.conf`).

### Variables

- **globalSession**: A pointer to a `Session` struct that stores the session in RAM for REPL/Stateless mode.

### Types

#### Session

The `Session` struct holds the following fields:

- **Username**: The username of the authenticated user.
- **Password**: The password of the authenticated user.
- **RawKey**: The raw key used for encryption/decryption.
- **Index**: The vault index associated with the session.

### Functions

#### SetGlobalSession

```go
func SetGlobalSession(s *Session)
```

Injects a session into RAM, used by the REPL.

#### Connect

```go
func Connect(username string, password string) error
```

Initializes the session and syncs the index locally. It takes the username and password as parameters and returns an error if the connection fails.

#### (s *Session) Save

```go
func (s *Session) Save() error
```

Saves the current session to the configuration file. Returns an error if the save operation fails.

#### GetSession

```go
func GetSession() (*Session, error)
```

Checks memory first (REPL cache) for an active session, then falls back to disk. Returns the session and an error if not connected.

#### Disconnect

```go
func Disconnect() error
```

Clears the memory cache and removes the configuration file.

#### FetchSessionStateless

```go
func FetchSessionStateless(username string, password string) (*Session, error)
```

Performs authentication and index fetch without saving to disk. It returns a session and an error if authentication fails.
