# Login-Go Service

A production-grade authentication and subscription management service written in Go. This service provides user authentication, subscription management, and monitoring capabilities.

## ⚠️ Production Readiness Notes

While this codebase has robust features, there are several critical items that need to be addressed for full production deployment:

1. **Database Configuration**: Currently hardcoded in `config/database.go`. Should be moved to environment variables.
2. **Missing Tests**: No automated tests are implemented. Need unit, integration and e2e tests.
3. **No API Versioning**: API endpoints should be versioned (e.g., /v1/users).
4. **Environment Configuration**: Needs a proper env configuration system.
5. **Backup Strategy**: Database backup procedure needs to be implemented.

## Features

- JWT-based authentication
- User management
- Subscription handling
- Rate limiting
- Prometheus metrics
- Health checks
- Graceful shutdown
- PostgreSQL database
- Input validation
- Error handling
- Logging with Zap

## Prerequisites

- Go 1.21+
- Docker and Docker Compose
- PostgreSQL

## Quick Start

1. Clone the repository
```bash
git clone https://github.com/JorgeSaicoski/login-go.git
```

2. Start services with Docker Compose
```bash
docker-compose up -d
```

The API will be available at http://localhost:8080

## API Routes

### Authentication
- `POST /auth/login` - User login
  ```json
  {
    "username": "string",
    "password": "string"
  }
  ```
- `POST /auth/validate` - Validate JWT token
  - Requires Authorization header with Bearer token

### Users
- `POST /user/register` - Create new user
  ```json
  {
    "name": "string",
    "username": "string",
    "email": "string",
    "password": "string"
  }
  ```
- `GET /user/:id` - Get user by ID
- `PATCH /user/:id` - Update user
  ```json
  {
    "name": "string",
    "email": "string"
  }
  ```

### Subscriptions
- `GET /subscription/:id` - Get subscription details
- `PATCH /subscription/:id` - Update subscription
  ```json
  {
    "name": "string",
    "description": "string",
    "price": number
  }
  ```

### User Subscriptions
- `GET /user/:userId/subscription` - Get user's subscriptions
- `POST /user/:userId/subscription/:subscriptionId` - Assign subscription to user
  ```json
  {
    "type": "individual|enterprise",
    "company_name": "string",
    "role": "string",
    "start_date": "datetime",
    "end_date": "datetime",
    "is_active": boolean
  }
  ```
- `PATCH /user/:userId/subscription/:subscriptionId` - Update user's subscription

### Health Checks
- `GET /health` - Service health check
- `GET /ready` - Service readiness check

## Security

- Rate limiting implemented
- Input validation
- Password hashing
- JWT token authentication
- Request timeouts
- Input sanitization

## Monitoring

- Prometheus metrics exposed
- Structured logging with Zap
- Health check endpoints

## Required Improvements for Production

### 1. Environment Configuration
Current:
```go
dsn := "host=db user=postgres password=yourpassword dbname=postgres port=5432 sslmode=disable"
```
Should be:
```go
dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
    os.Getenv("DB_HOST"),
    os.Getenv("DB_USER"),
    os.Getenv("DB_PASSWORD"),
    os.Getenv("DB_NAME"),
    os.Getenv("DB_PORT"))
```

### 2. API Versioning
Routes should be prefixed with version:
```go
v1 := r.Group("/v1")
{
    v1.POST("/auth/login", authHandler.Login)
    // ... other routes
}
```

### 3. Testing Strategy
Need to implement:
- Unit tests for business logic
- Integration tests for database operations
- End-to-end API tests
- Load tests

### 4. Backup Strategy
Implement:
- Regular database backups
- Backup verification
- Restore procedures
- Backup rotation policy

## Development

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

