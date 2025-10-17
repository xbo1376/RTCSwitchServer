# RTCSwitchServer

A small Go HTTP service that exposes simple room management endpoints for clients.

What it implements
- Create and destroy rooms via HTTP JSON APIs.
- Each room has: room ID, owner userid, creation time, room type (Live/Audio), and selected RTC service type (TRTC or Agora).
- RTC selection follows a 60:40 split logic implemented by mapping the last digit or a random selection (current code uses a 6:4 rule).

Build

From the project root you can build a local binary:

```bash
go build -o server_rtc_switch main.go
```

Cross-compile for Linux amd64 (works for pure-Go projects):

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server_rtc_switch .
```

Run

Start the server (default listens on :8376):

```bash
./server_rtc_switch
```

API

- POST /room/create
  - Request JSON: {"userid":"<user id>", "room_type":"Live|Audio"}
  - Response JSON: room object with fields: room_id, owner_userid, create_time, room_type, rtc_type

- POST /room/destroy
  - Request JSON: {"room_id":"<room id>"}
  - Response JSON: {"result":"ok"} on success

- GET /room/list
  - Returns a JSON array of rooms (sorted by creation time desc)

Examples

Create a room:

```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"userid":"5","room_type":"Live"}' \
  http://localhost:8376/room/create
```

List rooms:

```bash
curl http://localhost:8376/room/list
```

Notes
- Implementation stores rooms in memory (map + RWMutex). It's suitable for demo or single-process use. For production you should persist state and implement authentication/authorization.
- The RTC selection logic is deterministic/random within the server and can be adjusted to use the userid's last digit mapping if desired.

