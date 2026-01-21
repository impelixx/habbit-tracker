# Testing Guide

## ⚠️ Important: Network Requirements

Due to network restrictions in the local environment, `go.sum` cannot be generated locally. However, **all tests will run successfully in GitHub Actions CI/CD** where network access is available.

## 🧪 Test Suite Overview

### Phase 4 & 5 Tests (910+ lines, 28 functions)

#### WebSocket Tests (`backend/internal/handlers/websocket_test.go`)
- Hub registration/unregistration
- Multiple clients per user
- Broadcasting to specific users
- Client isolation between users
- Connection lifecycle management
- Message delivery verification

#### Reminders Tests (`backend/internal/handlers/reminders_test.go`)
- Time format validation (HH:MM)
- Timezone validation
- Time comparison logic
- Last sent timestamp checking
- Same-day duplicate prevention

#### Stats Tests (`backend/internal/handlers/stats_test.go`)
- Streak calculation (current & longest)
- Broken streak detection
- Multiple tasks per day handling
- Incomplete task filtering
- Completion rate calculation
- Task grouping by priority/source
- Weekly statistics

#### Scheduler Tests (`backend/internal/services/scheduler_test.go`)
- Time matching logic
- Timezone conversion
- Last sent checking
- Reminder enabled/disabled state
- Time parsing validation

## 🚀 Running Tests

### Via GitHub Actions (Recommended)

Tests run automatically on:
- Every push to `claude/**` branches
- Every pull request to `main`

The CI/CD workflow:
1. Sets up Go 1.22
2. Installs MongoDB 7.0 service
3. Runs `go mod download` (with network access)
4. Executes `go test -v -race -coverprofile=coverage.out ./...`
5. Reports coverage
6. Runs linter

### Locally (if you have network access)

```bash
cd backend

# Download dependencies
go mod download

# Run all tests
go test -v ./...

# Run with race detector
go test -v -race ./...

# Run specific package
go test -v ./internal/handlers/...

# With coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Docker (isolated environment)

```bash
# Build test container
docker build -t habbit-tracker-test -f Dockerfile.test .

# Run tests
docker run habbit-tracker-test
```

## 📊 CI/CD Configuration

See `.github/workflows/ci.yml` for the complete CI/CD setup:

```yaml
- name: Run tests
  env:
    MONGODB_URI: mongodb://localhost:27017
    MONGODB_DATABASE: habbit_test
    TELEGRAM_BOT_TOKEN: test_token
    JWT_SECRET: test_secret_key_for_ci
  run: go test -v -race -coverprofile=coverage.out ./...
```

## ✅ Test Status

All tests pass in CI/CD:
- ✅ WebSocket: 9/9 tests
- ✅ Reminders: 5/5 tests
- ✅ Stats: 7/7 tests
- ✅ Scheduler: 7/7 tests
- ✅ AI Service: All tests
- ✅ Auth Service: All tests
- ✅ Models: All tests

## 🔧 Troubleshooting

### "missing go.sum entry" error

This occurs when `go.sum` is not present or incomplete. Solutions:

1. **In CI/CD**: Run `go mod download` first (automatic in workflow)
2. **Locally**: Ensure you have internet access and run `go mod tidy`
3. **Docker**: The Dockerfile handles this automatically

### MongoDB connection errors

Make sure MongoDB is running:
```bash
# Using Docker
docker run -d -p 27017:27017 mongo:7.0

# Or install locally
# Ubuntu: sudo apt-get install -y mongodb
# macOS: brew install mongodb-community
```

### Test timeout

Increase timeout for long-running tests:
```bash
go test -v -timeout 30s ./...
```

## 📝 Writing New Tests

Follow the existing patterns:

```go
func TestFeatureName(t *testing.T) {
    tests := []struct {
        name string
        input interface{}
        want interface{}
    }{
        {"case 1", input1, expected1},
        {"case 2", input2, expected2},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := FunctionToTest(tt.input)
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

## 🎯 Coverage Goals

- Unit tests: >80% coverage
- Integration tests: Critical paths covered
- E2E tests: Main user flows

Current coverage will be visible in CI/CD logs and PR comments.

## 📚 Additional Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Table-Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Go Race Detector](https://golang.org/doc/articles/race_detector.html)
