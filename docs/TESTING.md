# Testing Guide

## Phase 1: Testing the Telegram Bot

### Prerequisites

- Telegram account
- Go 1.22+ installed
- Docker and Docker Compose installed
- ngrok installed (or any other tunneling tool)

### Step-by-Step Testing

#### 1. Create Telegram Bot

1. Open Telegram and search for `@BotFather`
2. Send `/newbot` command
3. Follow the prompts:
   - Choose a name for your bot (e.g., "My Habit Tracker")
   - Choose a username (must end in 'bot', e.g., "my_habit_tracker_bot")
4. Save the bot token (format: `123456789:ABCdefGHIjklMNOpqrsTUVwxyz`)

#### 2. Set Bot Commands

Send this to @BotFather to configure commands:

```
/mybots
[Select your bot]
Edit Bot > Edit Commands

Then paste:
start - Start using the bot
today - View today's tasks
add - Add a new task
done - Mark tasks as completed
settings - Configure reminders
help - Show help message
```

#### 3. Configure Environment

```bash
cd backend
cp .env.example .env
```

Edit `.env` and set:
```env
TELEGRAM_BOT_TOKEN=your-bot-token-here
MONGODB_URI=mongodb://localhost:27017
```

#### 4. Start MongoDB

```bash
# From project root
docker-compose up -d mongodb

# Verify MongoDB is running
docker-compose ps
```

#### 5. Run Backend

**Option A: Direct with Go**
```bash
cd backend
go mod download  # Download dependencies
go run cmd/server/main.go
```

**Option B: Using Docker Compose**
```bash
# From project root
docker-compose up backend
```

You should see:
```
Starting Habit Tracker Bot...
Connected to MongoDB successfully
Authorized on Telegram bot account: YourBotName
Starting server on :8080
```

#### 6. Expose Webhook with ngrok

In a new terminal:
```bash
ngrok http 8080
```

Copy the HTTPS URL (e.g., `https://abc123.ngrok.io`)

#### 7. Set Telegram Webhook

```bash
curl -X POST "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook" \
  -d "url=https://abc123.ngrok.io/api/webhook/telegram"
```

Expected response:
```json
{
  "ok": true,
  "result": true,
  "description": "Webhook was set"
}
```

Verify webhook:
```bash
curl "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getWebhookInfo"
```

#### 8. Test Bot Commands

Open Telegram and find your bot. Test these commands:

**Test 1: Start**
```
/start
```
Expected: Welcome message with command list

**Test 2: Add Task**
```
/add Morning workout
```
Expected: "✅ Task added: Morning workout"

Or just send plain text:
```
Read 30 pages
```
Expected: "✅ Task added: Read 30 pages"

**Test 3: View Today's Tasks**
```
/today
```
Expected: List of tasks with status (⭕ incomplete, ✅ completed)

**Test 4: Complete Tasks**
```
/done
```
Expected: Interactive buttons to mark tasks as complete

Click a button → Task marked as completed

**Test 5: Help**
```
/help
```
Expected: List of all commands and features

**Test 6: Settings**
```
/settings
```
Expected: Settings placeholder message

### Verification Checklist

- [ ] Bot responds to `/start`
- [ ] Can add tasks with `/add <task>`
- [ ] Can add tasks by sending plain text
- [ ] `/today` shows today's tasks
- [ ] `/done` shows interactive buttons
- [ ] Clicking button marks task as completed
- [ ] `/help` shows command list
- [ ] Tasks persist (visible after bot restart)
- [ ] Multiple tasks can be managed

### Check Database

To verify data is saved in MongoDB:

```bash
# Connect to MongoDB
docker exec -it habbit-tracker-mongo mongosh

# Switch to database
use habbit

# View users
db.users.find().pretty()

# View tasks
db.tasks.find().pretty()

# Count tasks
db.tasks.countDocuments()
```

### Troubleshooting

**Bot doesn't respond:**
1. Check backend logs for errors
2. Verify webhook is set: `curl https://api.telegram.org/bot<TOKEN>/getWebhookInfo`
3. Check ngrok is running
4. Verify MongoDB connection in backend logs

**"Failed to connect to MongoDB":**
- Ensure MongoDB is running: `docker-compose ps`
- Check MongoDB URI in `.env`
- Try: `docker-compose restart mongodb`

**"Failed to set webhook":**
- Webhook URL must be HTTPS
- ngrok must be running
- Check if bot token is correct

**Tasks not persisting:**
- Check MongoDB connection
- Verify database name in `.env` (should be `habbit`)
- Check backend logs for database errors

### Performance Testing

Test with multiple tasks:

```bash
# Add 10 tasks quickly
/add Task 1
/add Task 2
/add Task 3
...
/add Task 10

# View all
/today

# Complete several
/done
```

### Logs

Check backend logs for:
- Request/response logging
- Database operations
- Error messages

Example log output:
```
2026-01-11 10:30:45 200 - POST /api/webhook/telegram (5ms)
New user created telegramId=123456789
Task created: Morning workout
```

### Clean Up

Stop services:
```bash
docker-compose down

# Remove volumes (deletes all data)
docker-compose down -v
```

## Next: Phase 2 Testing

Once Phase 2 (Mini App) is implemented, you'll be able to:
- Open the Mini App from bot
- View tasks in a rich UI
- Create/edit/delete tasks visually
- See real-time updates

Testing guide for Phase 2 will be added after implementation.
