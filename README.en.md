# SwiftPost Email System

SwiftPost is a feature-rich email system incorporating all the essential functionalities required by modern email services. The system uses Go as the primary backend framework, Python for auxiliary services, and HTML/CSS/JavaScript for the frontend, with WebSocket support for real-time communication.

## Key Features

- **User System**: Registration, login, profile management, custom domain support
- **Email Management**: Send/receive emails, categorize emails, mark read/unread status, star important messages
- **Real-time Notifications**: Instant email alerts via WebSocket
- **Admin Panel**: User management, email management, system monitoring, log viewing
- **System Monitoring**: Real-time performance metrics including user statistics, email statistics, and storage usage
- **Multi-language Support**: Automatic switching between Chinese and English interfaces
- **Security Features**: HTTPS support, password strength validation, access control

## Technical Architecture

- **Backend**: Implemented in Go, using the Mux routing library to provide RESTful APIs
- **Database**: SQLite as the default storage engine
- **Frontend**: Responsive design with mobile device support
- **Real-time Communication**: WebSocket-based instant notification system
- **Deployment**: Containerized deployment via Docker for quick installation and configuration

## Installation & Deployment

Refer to the `install.sh` script in the project for automated installation, or deploy using Docker.

## Directory Structure

```
├── backend/          # Backend service code
│   ├── go/           # Core services implemented in Go
│   └── python/       # Auxiliary Python services
├── frontend/         # Frontend pages and assets
│   ├── static/       # Static resources (CSS/JS)
│   └── templates/    # HTML templates
├── config.json       # Configuration file
├── Dockerfile        # Docker build file
├── nginx/            # Nginx configuration files
└── systemd/          # System service configuration files
```

## API Documentation

Complete API documentation can be found in the `handlers` directory, including the following main endpoints:

- `/api/auth/` Authentication endpoints (registration, login, logout, etc.)
- `/api/email/` Email-related endpoints (send, retrieve, manage emails)
- `/api/user/` User management endpoints
- `/api/admin/` Administrator-only endpoints
- `/api/stats/` Statistics endpoints
- `/api/health` Health check endpoint

## Contribution Guidelines

Contributions are welcome! Please follow these steps:
1. Fork the repository
2. Create a new branch
3. Commit your changes
4. Submit a Pull Request

## License

This project is licensed under the MIT License. See the LICENSE file for details.