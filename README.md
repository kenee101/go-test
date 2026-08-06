# Task Management REST API (Go)

Simple REST API in Go with MongoDB and JWT authentication.

Run locally:

1. Start MongoDB (or use a hosted instance) and set `MONGO_URI`/`MONGO_DB` in the environment.
2. Set `JWT_SECRET` environment variable.
3. Build and run:

```bash
go run main.go
```

Endpoints:
- `POST /register` — register new user
- `POST /login` — login and receive JWT
- `POST /tasks` — create task (auth)
- `GET /tasks` — list tasks for user (auth)
- `GET /tasks/{id}` — get single task (auth)
- `PUT /tasks/{id}` — update task (auth)
- `DELETE /tasks/{id}` — delete task (auth)
- `GET /admin/tasks` — list all tasks (admin only)
