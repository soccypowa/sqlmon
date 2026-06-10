package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	_ "github.com/microsoft/go-mssqldb"
)

type Session struct {
	SPID         int
	Status       string
	LoginName    string
	HostName     string
	ProgramName  string
	DatabaseName string
	WaitType     string
	WaitTimeMs   int64
	CPUTime      int64
	ElapsedMs    int64
	BlockedBy    int
	QueryText    string
}

type Client struct {
	db         *sql.DB
	ServerName string // displayname set from the connection string
}

// New opens a connection pool to SQL Server and verifies it with
// a ping. Returns an error if the server is unreachable or the
// credentials are wrong.
//
// CONNECTION STRING FORMAT
//
//	sqlserver://user:password@host[:port][/instance][?param=value]
//
// Examples:
//
//	sqlserver://sa:pass@localhost
//	sqlserver://sa:pass@myserver:1433
//	sqlserver://myserver?trusted_connection=true   (Windows auth)

func New(connStr string) (*Client, error) {
	u, err := url.Parse(connStr)
	if err != nil {
		return nil, fmt.Errorf("invalid connection string: %v", err)
	}
	q := u.Query()
	if q.Get("app name") == "" {
		q.Set("app name", "sqlmon")
	}
	u.RawQuery = q.Encode()
	connStr = u.String()

	db, err := sql.Open("sqlserver", connStr)
	if err != nil {
		return nil, fmt.Errorf("could not open connection: %v", err)
	}

	// To actulaly open the connection
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping failed: %v", err)
	}

	// Keep the connpool small, we are not an app server
	db.SetMaxOpenConns(3)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(60 * time.Second)

	return &Client{
		db: db,
	}, nil
}

func (c *Client) Close() {
	c.db.Close()
}

// ------------------------------------------------------------------
// GetSessions — the only query in this lesson
// ------------------------------------------------------------------
//
// Joins sys.dm_exec_sessions (one row per connection) with
// sys.dm_exec_requests (one row per active request) and
// sys.dm_exec_sql_text (the actual SQL being run, via OUTER APPLY).
//
// OUTER APPLY means: for each session, call dm_exec_sql_text with
// the request's sql_handle. If there is no active request (session
// is idle), the row still appears but sql_handle is NULL → text
// comes back NULL → we ISNULL it to an empty string.
//
// WHY ISNULL EVERYWHERE?
//   dm_exec_sessions.host_name can be NULL for system sessions.
//   Scanning a NULL into a Go string panics unless you use
//   sql.NullString. Wrapping with ISNULL(..., '') at the SQL level
//   is simpler and keeps the Go code clean.
//
// REQUIRES: VIEW SERVER STATE permission on the login.
//   GRANT VIEW SERVER STATE TO [yourlogin];

func (c *Client) GetSessions(ctx context.Context) ([]Session, error) {
	const query = `
SELECT
    s.session_id,
    ISNULL(s.status,       '')  AS status,
    ISNULL(s.login_name,   '')  AS login_name,
    ISNULL(s.host_name,    '')  AS host_name,
    ISNULL(s.program_name, '')  AS program_name,
    ISNULL(DB_NAME(s.database_id), '') AS database_name,
    ISNULL(r.wait_type,    '')  AS wait_type,
    ISNULL(r.wait_time,     0)  AS wait_time_ms,
    ISNULL(r.cpu_time,      0)  AS cpu_time,
    ISNULL(r.total_elapsed_time, 0) AS elapsed_ms,
    ISNULL(r.blocking_session_id, 0) AS blocked_by,
    ISNULL(SUBSTRING(t.text, 1, 200), '') AS query_text
FROM sys.dm_exec_sessions   AS s
LEFT  JOIN sys.dm_exec_requests AS r  ON s.session_id = r.session_id
OUTER APPLY sys.dm_exec_sql_text(r.sql_handle) AS t
WHERE s.is_user_process = 1
ORDER BY ISNULL(r.total_elapsed_time, 0) DESC`

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query sessions failed: %v", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		err := rows.Scan(
			&s.SPID,
			&s.Status,
			&s.LoginName,
			&s.HostName,
			&s.ProgramName,
			&s.DatabaseName,
			&s.WaitType,
			&s.WaitTimeMs,
			&s.CPUTime,
			&s.ElapsedMs,
			&s.BlockedBy,
			&s.QueryText,
		)
		if err != nil {
			// Skip malformed rows rather than aborting the whole resulr
			continue
		}
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %v", err)
	}

	return sessions, nil
}
