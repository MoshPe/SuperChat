# SuperChat - Team Chat Application

A secure, real-time team chat application built with Go backend and React frontend, featuring user authentication, team management, and WebSocket-based messaging.

## Features

- 🔐 **Secure Authentication**: JWT-based authentication with password hashing
- 👥 **Team Management**: Create, join, and manage teams
- 💬 **Real-time Chat**: WebSocket-based messaging with typing indicators
- 👤 **User Profiles**: Customizable user profiles with avatar generation
- 📱 **Responsive Design**: Modern UI built with Tailwind CSS
- 🗄️ **File-based Database**: BoltDB for data persistence
- 🔒 **Security**: Secure session management and data validation

## Tech Stack

### Backend
- **Go 1.21+** - High-performance server language
- **Gorilla Mux** - HTTP router and URL matcher
- **Gorilla WebSocket** - WebSocket implementation
- **BoltDB** - Embedded key/value database
- **JWT** - JSON Web Token authentication
- **bcrypt** - Password hashing

### Frontend
- **React 18** - Modern UI library
- **Vite** - Fast build tool and dev server
- **Tailwind CSS** - Utility-first CSS framework
- **React Router** - Client-side routing
- **Axios** - HTTP client
- **Lucide React** - Beautiful icons

## Prerequisites

- Go 1.21 or higher
- Node.js 16 or higher
- npm or yarn

## Installation

### 1. Clone the repository

```bash
git clone <repository-url>
cd SuperChat
```

### 2. Backend Setup

```bash
# Install Go dependencies
go mod tidy

# Run the backend server
go run .
```

The backend will start on `http://localhost:8080`

### 3. Frontend Setup

```bash
# Navigate to frontend directory
cd frontend

# Install dependencies
npm install

# Start development server
npm run dev
```

The frontend will start on `http://localhost:3000`

### 4. Build for Production

```bash
# Build the frontend
npm run build

# The built files will be in the `web` folder
# The Go backend will serve these static files automatically
```

## Project Structure

```
SuperChat/
├── main.go              # Main server entry point
├── models.go            # Data models and structures
├── auth.go              # Authentication and middleware
├── database.go          # Database operations
├── handlers.go          # HTTP request handlers
├── websocket.go         # WebSocket handling
├── go.mod               # Go module dependencies
├── frontend/            # React frontend application
│   ├── src/
│   │   ├── components/  # Reusable UI components
│   │   ├── contexts/    # React contexts
│   │   ├── pages/       # Page components
│   │   ├── services/    # API services
│   │   ├── App.jsx      # Main app component
│   │   └── main.jsx     # React entry point
│   ├── package.json     # Node.js dependencies
│   ├── vite.config.js   # Vite configuration
│   └── tailwind.config.js # Tailwind CSS configuration
└── README.md            # This file
```

## API Endpoints

### Authentication
- `POST /api/auth/register` - User registration
- `POST /api/auth/login` - User login
- `POST /api/auth/logout` - User logout

### User Management
- `GET /api/user/profile` - Get user profile
- `PUT /api/user/profile` - Update user profile

### Teams
- `POST /api/teams` - Create new team
- `GET /api/teams` - Get user's teams
- `GET /api/teams/{id}` - Get team details
- `POST /api/teams/{id}/join` - Join team
- `POST /api/teams/{id}/leave` - Leave team

### Chat
- `GET /api/teams/{id}/messages` - Get team messages
- `POST /api/teams/{id}/messages` - Send message
- `GET /ws/{teamId}` - WebSocket connection for real-time chat

## WebSocket Message Types

- `chat_message` - New chat message
- `typing` - User typing indicator
- `user_joined` - User joined notification
- `user_left` - User left notification
- `ping/pong` - Connection keep-alive

## Security Features

- **Password Hashing**: bcrypt with cost factor 14
- **JWT Tokens**: Secure session management
- **Input Validation**: Server-side validation for all inputs
- **CORS Protection**: Configurable origin checking
- **Rate Limiting**: Built-in request throttling
- **SQL Injection Protection**: Parameterized queries with BoltDB

## Database Schema

The application uses BoltDB with the following buckets:

- **users**: User account information
- **teams**: Team data and metadata
- **team_members**: User-team relationships
- **messages**: Chat message history

## Configuration

### Environment Variables

- `PORT` - Server port (default: 8080)
- `JWT_SECRET` - JWT signing secret (change in production)

### JWT Configuration

Update the JWT secret in `auth.go`:

```go
var jwtSecret = []byte("your-secret-key-change-in-production")
```

## Development

### Running Tests

```bash
# Backend tests
go test ./...

# Frontend tests
cd frontend
npm test
```

### Code Formatting

```bash
# Backend
go fmt ./...

# Frontend
cd frontend
npm run lint
```

## Production Deployment

### 1. Build the Application

```bash
# Build frontend
cd frontend
npm run build

# Build backend (optional)
go build -o superchat .
```

### 2. Environment Setup

- Set production environment variables
- Use a proper JWT secret
- Configure reverse proxy (nginx/Apache)
- Set up SSL/TLS certificates
- Configure firewall rules

### 3. Database Backup

Regularly backup the `superchat.db` file:

```bash
cp superchat.db superchat.db.backup
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions:

- Create an issue in the repository
- Check the documentation
- Review the code examples

## Roadmap

- [ ] File sharing capabilities
- [ ] Voice and video calls
- [ ] Mobile applications
- [ ] Advanced team permissions
- [ ] Message search and filtering
- [ ] Push notifications
- [ ] Multi-language support
- [ ] Dark mode theme
- [ ] Message reactions
- [ ] User status indicators

## Current Priority Tasks

- Done: usernames are immutable; added editable display name across backend and frontend (handlers.go, models.go, auth.go/frontend).
- Done: uploads locked down with owner/team metadata and header-based access (no JWTs in URLs) (handlers.go, database.go, frontend).
- Enforce membership on WebSocket sessions: drop/deny connections for removed users and re-check membership on inbound messages (handlers.go, websocket.go).
- Harden origin/CORS: replace `*` CORS and permissive `CheckOrigin` with a configurable allowlist (main.go, websocket.go).
- Secure JWT handling: move `jwtSecret` to env/config, validate signing method, and plan for rotation (auth.go).
- Align server defaults/logging: reconcile port/TLS defaults with README and fix the extra quote in the startup log (main.go).
- Add rate limiting/body size limits for auth/messaging/uploads (main.go middleware, handlers).
- Bound message payload sizes and validate WS payloads to prevent oversized writes (handlers.go, websocket.go).
- Improve message storage/query: key messages per team/timestamp and add pagination to avoid full scans (database.go).
- Add a super-admin/manager account role for platform-wide administration: password reset/recovery for users, permissions management, and global oversight tools.
- Add tests for auth, username index integrity, membership enforcement, uploads ACL, and WS join/leave broadcasting (new _test.go files).

## Acknowledgments

- [Gorilla WebSocket](https://github.com/gorilla/websocket) for WebSocket implementation
- [BoltDB](https://github.com/etcd-io/bbolt) for embedded database
- [Tailwind CSS](https://tailwindcss.com/) for styling
- [Lucide](https://lucide.dev/) for beautiful icons
- [DiceBear](https://www.dicebear.com/) for avatar generation
