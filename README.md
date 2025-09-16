# Blog Aggregator (Gator)

A command-line RSS feed aggregator built with Go that allows you to follow, manage, and browse multiple blog feeds from a single interface.
> **Note**: This project was built as part of a guided learning experience from [boot.dev](https://boot.dev), following their Go backend development course.
## 🚀 Features

- **User Management**: Register users and manage login sessions
- **Feed Management**: Add, follow, and unfollow RSS/Atom feeds
- **Post Aggregation**: Automatically fetch and store posts from followed feeds
- **Browse Posts**: View recent posts from your followed feeds
- **Database Persistence**: All data stored in PostgreSQL with proper schema migrations

## 🛠️ Technologies Used

- **[Go](https://golang.org/)** - Primary programming language
- **[PostgreSQL](https://www.postgresql.org/)** - Database for storing users, feeds, and posts
- **[SQLC](https://github.com/sqlc-dev/sqlc)** - Generate type-safe Go code from SQL queries
- **[Goose](https://github.com/pressly/goose)** - Database migration tool
- **[lib/pq](https://github.com/lib/pq)** - PostgreSQL driver for Go
- **[UUID](https://github.com/google/uuid)** - Generate unique identifiers

## 📋 Prerequisites

Before running this application, ensure you have the following installed:

1. **Go 1.24.0 or later**
   - Download from: https://golang.org/dl/
   - Verify installation: `go version`

2. **PostgreSQL**
   - Download from: https://www.postgresql.org/download/
   - Ensure PostgreSQL service is running
   - Create a database for the application

3. **Git** (for cloning the repository)

## 🔧 Installation

### 1. Install the Gator CLI

```bash
go install github.com/max-durnea/blog-aggregator@latest
```

This will install the `blog-aggregator` binary to your `$GOPATH/bin` directory. Make sure this directory is in your system's PATH.

### 2. Alternative: Build from Source

```bash
# Clone the repository
git clone https://github.com/max-durnea/blog-aggregator.git
cd blog-aggregator

# Build the application
go build -o gator .

# Optionally, move to a directory in your PATH
mv gator /usr/local/bin/  # Linux/macOS
# or move to a directory in your PATH on Windows
```

## ⚙️ Configuration

### 1. Create Configuration File

Create a `.gatorconfig.json` file in your home directory:

```json
{
    "db_url": "postgres://username:password@localhost:5432/database_name?sslmode=disable",
    "current_user_name": ""
}
```

**Configuration Parameters:**
- `db_url`: PostgreSQL connection string
- `current_user_name`: Currently logged-in user (managed by the application)

### 2. Database Setup

Set up your PostgreSQL database and run migrations:

```bash
# Install goose for migrations (if not already installed)
go install github.com/pressly/goose/v3/cmd/goose@latest

# Run database migrations
goose -dir sql/schema postgres "your-connection-string" up
```

Example connection string:
```
postgres://myuser:mypassword@localhost:5432/gator_db?sslmode=disable
```

## 🎮 Usage

### Basic Commands

#### User Management

```bash
# Register a new user
./gator register <username>

# Login as an existing user
./gator login <username>

# List all registered users
./gator users

# Reset all users (⚠️ destructive operation)
./gator reset
```

#### Feed Management

```bash
# Add a new RSS feed
./gator addfeed <feed_name> <feed_url>

# List all feeds
./gator feeds

# Follow a feed (by URL)
./gator follow <feed_url>

# List feeds you're following
./gator following

# Unfollow a feed
./gator unfollow <feed_url>
```

#### Content Browsing

```bash
# Browse recent posts from followed feeds
./gator browse [limit]

# Start automatic feed aggregation (fetches feeds periodically)
./gator agg <duration>
# Examples:
./gator agg 30s    # Every 30 seconds
./gator agg 5m     # Every 5 minutes
./gator agg 1h     # Every hour
```

### Example Workflow

```bash
# 1. Register and login
./gator register alice
./gator login alice

# 2. Add and follow some feeds
./gator addfeed "Tech Blog" "https://example.com/feed.xml"
./gator follow "https://example.com/feed.xml"

# 3. Start aggregation to fetch posts
./gator agg 1m

# 4. In another terminal, browse posts
./gator browse 10
```

## 📁 Project Structure

```
blog-aggregator/
├── main.go                 # Application entry point
├── handlers.go            # Command handlers and RSS feed processing
├── go.mod                 # Go module dependencies
├── sqlc.yaml             # SQLC configuration
├── internal/
│   ├── config/           # Configuration management
│   └── database/         # Generated database code (SQLC)
└── sql/
    ├── schema/           # Database migrations (Goose)
    │   ├── 001_users.sql
    │   ├── 002_feeds.sql
    │   ├── 003_feed_follows.sql
    │   ├── 004_add_last_fetched.sql
    │   └── 005_posts.sql
    └── queries/          # SQL queries (SQLC input)
        ├── users.sql
        ├── feed.sql
        ├── feed_follows.sql
        └── posts.sql
```

## 🗄️ Database Schema

The application uses the following main tables:

- **users**: User accounts and authentication
- **feeds**: RSS/Atom feed information
- **feed_follows**: Many-to-many relationship between users and feeds
- **posts**: Individual blog posts fetched from feeds

## 🔄 Development

### Regenerating Database Code

If you modify SQL queries or schema:

```bash
# Install sqlc (if not already installed)
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Regenerate Go code from SQL
sqlc generate
```

### Adding New Migrations

```bash
# Create a new migration file
goose -dir sql/schema create add_new_feature sql

# Edit the generated file, then run:
goose -dir sql/schema postgres "connection-string" up
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and ensure code builds
5. Submit a pull request

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🐛 Troubleshooting

### Common Issues

1. **"Command not found"**: Ensure `$GOPATH/bin` is in your PATH
2. **Database connection errors**: Verify PostgreSQL is running and connection string is correct
3. **Migration errors**: Ensure you have proper database permissions
4. **Feed parsing errors**: Some feeds may have malformed XML - check feed URL validity

### Debug Mode

For verbose output during feed fetching, the application includes debug printing to help troubleshoot RSS feed parsing issues.

## 📚 Further Reading

- [Go Documentation](https://golang.org/doc/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [SQLC Documentation](https://docs.sqlc.dev/)
- [Goose Migration Tool](https://github.com/pressly/goose)

