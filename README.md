# My sandbox to fun with AI agents to write code
# Habit Tracker - Telegram Bot + Mini App

A comprehensive habit tracking system built with Telegram Bot + Mini App, featuring AI-powered voice recognition and natural language task parsing.

## Features

- 🤖 **Telegram Bot**: Interact via text commands, voice messages, and inline buttons
- 📱 **Mini App**: Rich React UI integrated into Telegram
- 🎤 **Voice Recognition**: Dictate your plans and get structured to-do lists
- 🔔 **Smart Reminders**: Daily notifications with timezone support
- 🔄 **Real-time Sync**: WebSocket connection keeps everything in sync
- 🧠 **AI-Powered**: Natural language parsing using OpenRouter free models
- ☁️ **Free Hosting**: Runs on Railway + Vercel + MongoDB Atlas free tiers

## Tech Stack

### Backend
- **Go 1.22** - Main language
- **Fiber v2** - Web framework (fast, low memory)
- **MongoDB** - Database (Atlas M0 free tier)
- **Telegram Bot API** - Bot interactions
- **OpenRouter** - AI models (LLM for task parsing)

### Frontend
- **React 18** - UI library
- **TypeScript** - Type safety
- **Vite** - Build tool
- **Telegram WebApp SDK** - Mini App integration

## Architecture

```
┌─────────────┐
│ Telegram    │
│ User        │
└──────┬──────┘
       │
       ├───────────────┐
       │               │
       v               v
┌──────────┐    ┌──────────────┐
│ Bot      │    │ Mini App     │
│ Commands │    │ (React UI)   │
└────┬─────┘    └──────┬───────┘
     │                 │
     │ Webhook         │ REST + WS
     v                 v
┌────────────────────────────────┐
│  Backend (Go/Fiber)            │
│  - REST API                    │
│  - WebSocket Hub               │
│  - Telegram Webhook Handler    │
│  - AI Service (OpenRouter)     │
│  - Reminder Scheduler          │
└────────────┬───────────────────┘
             │
             v
┌─────────────────────────────────┐
│  MongoDB Atlas M0               │
│  - users                        │
│  - tasks                        │
│  - reminders                    │
└─────────────────────────────────┘
```

## Project Structure

```
habbit-tracker/
├── backend/
│   ├── cmd/server/
│   │   └── main.go              # Entry point
│   ├── internal/
│   │   ├── config/             # Configuration
│   │   ├── db/                 # MongoDB operations
│   │   ├── handlers/           # HTTP/Webhook handlers
│   │   ├── middleware/         # Middleware (auth, rate limit)
│   │   ├── models/             # Data models
│   │   ├── services/           # Business logic
│   │   └── utils/              # Utilities
│   ├── Dockerfile
│   ├── go.mod
│   └── .env.example
├── frontend/
│   ├── src/
│   │   ├── components/         # React components
│   │   ├── hooks/              # Custom hooks
│   │   ├── services/           # API clients
│   │   └── types/              # TypeScript types
│   ├── package.json
│   └── vite.config.ts
├── docker-compose.yml
└── README.md
```

## Getting Started

### Prerequisites

- Go 1.22+
- Node.js 20+
- Docker & Docker Compose
- ngrok (for local Telegram webhook testing)
- Telegram Bot Token (from @BotFather)
- MongoDB Atlas account (free M0 cluster)
- OpenRouter API key (optional, for AI features)

### Local Development Setup

#### 1. Clone the repository

```bash
git clone <repository-url>
cd habbit-tracker
```

#### 2. Set up MongoDB

Start MongoDB locally using Docker Compose:

```bash
docker-compose up -d mongodb
```

Or use MongoDB Atlas (recommended):
1. Create a free M0 cluster at https://www.mongodb.com/cloud/atlas
2. Get connection string
3. Add to `.env` file

#### 3. Configure Backend

```bash
cd backend
cp .env.example .env
# Edit .env with your credentials
```

Required environment variables:
- `TELEGRAM_BOT_TOKEN` - Get from @BotFather
- `MONGODB_URI` - Your MongoDB connection string
- `JWT_SECRET` - Random secret key (change in production)
- `OPENROUTER_API_KEY` - Get from https://openrouter.ai (optional)

#### 4. Run Backend

**Option A: Using Go directly**
```bash
cd backend
go mod download
go run cmd/server/main.go
```

**Option B: Using Docker Compose**
```bash
docker-compose up backend
```

Run full stack (MongoDB + Backend + Frontend):
```bash
docker-compose up -d
```

Frontend will be available at `http://localhost:3000`

#### 5. Set up Telegram Webhook (for local testing)

In a separate terminal, expose your local server:

```bash
ngrok http 8080
```

Copy the ngrok URL (e.g., `https://abc123.ngrok.io`) and set the webhook:

```bash
curl -X POST "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook" \
  -d "url=https://abc123.ngrok.io/api/webhook/telegram"
```

#### 6. Test the Bot

Open Telegram and send `/start` to your bot!

### Bot Commands

- `/start` - Start using the bot
- `/help` - Show help message
- `/add <task>` - Add a new task
- `/today` - View today's tasks
- `/done` - Mark tasks as completed
- `/settings` - Configure reminders

### Frontend Development (Mini App)

```bash
cd frontend
npm install
cp .env.example .env
# Edit .env with backend URL
npm run dev
```

The Mini App will be available at `http://localhost:5173`

## Deployment

### Backend (Railway)

1. Create Railway project: https://railway.app
2. Connect GitHub repository
3. Set environment variables (see `.env.example`)
4. Deploy from `backend/` directory
5. Copy Railway URL
6. Set Telegram webhook to Railway URL

### Frontend (Vercel)

1. Connect GitHub to Vercel: https://vercel.com
2. Set root directory to `frontend/`
3. Set environment variables:
   - `VITE_API_URL=https://your-app.railway.app`
   - `VITE_WS_URL=wss://your-app.railway.app`
4. Deploy
5. Configure Mini App in @BotFather using Vercel URL

### MongoDB Atlas

1. Create free M0 cluster
2. Add network access: `0.0.0.0/0` (for Railway)
3. Create database user
4. Get connection string
5. Add to Railway environment variables

## API Documentation

### Endpoints

- `GET /health` - Health check
- `POST /api/webhook/telegram` - Telegram webhook
- `POST /api/auth/verify` - Authenticate Mini App user
- `GET /api/tasks` - Get tasks
- `POST /api/tasks` - Create task
- `PATCH /api/tasks/:id` - Update task
- `DELETE /api/tasks/:id` - Delete task
- `GET /api/stats` - Get statistics
- `POST /api/reminders/set` - Configure reminders
- `WS /api/ws` - WebSocket for real-time updates

See [docs/API.md](docs/API.md) for detailed API documentation.

## Development Roadmap

### Phase 1: Core Backend + Bot ✅ (Current)
- [x] Go project structure with Fiber
- [x] MongoDB models and connection
- [x] Telegram bot with basic commands
- [x] Docker setup
- [ ] Basic task CRUD

### Phase 2: Mini App + Auth (Next)
- [ ] React Mini App skeleton
- [ ] Telegram WebApp SDK integration
- [ ] Authentication flow (initData → JWT)
- [ ] Task list UI
- [ ] REST API client

### Phase 3: AI Integration
- [ ] OpenRouter integration
- [ ] Voice message handling
- [ ] Natural language task parsing
- [ ] STT via Telegram API

### Phase 4: Reminders
- [ ] Background scheduler
- [ ] Timezone-aware reminders
- [ ] Daily task notifications
- [ ] Reminder settings UI

### Phase 5: WebSocket Sync
- [ ] WebSocket hub
- [ ] Real-time task updates
- [ ] Multi-device sync
- [ ] Auto-reconnection

### Phase 6: Production
- [ ] Rate limiting
- [ ] Caching layer
- [ ] Monitoring & logging
- [ ] Full deployment

## Contributing

Contributions are welcome! Please follow these guidelines:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details

## Support

For issues and questions:
- Open an issue on GitHub
- Contact: [Your contact info]

## Acknowledgments

- Telegram Bot API
- OpenRouter for free AI models
- Railway, Vercel, MongoDB Atlas for free hosting
