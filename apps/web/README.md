# Pizza Console

Web console for Pizza auth-service and control-plane HTTP API.

## Features

- Register, login, verify email, forgot/reset password
- Session (/me), change password, logout + refresh
- Organizations: list/create
- Org detail: members (add/role/remove), projects, API keys (create once + revoke)

## Run

```bash
cp .env.example .env
npm install
npm run dev
```

http://localhost:3000

## Env

- `VITE_AUTH_URL` — auth-service base (Northflank)
- `VITE_CONTROL_URL` — control-plane HTTP API (default http://localhost:8081)

## Backend

- Auth: already deployed
- Control-plane: `HTTP_PORT=8081` + `JWT_SECRET` same as auth + `DATABASE_URL` + migrations
- CORS on both services must allow `http://localhost:3000`
