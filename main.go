package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// RTCType represents the underlying RTC service type
type RTCType string

const (
	RTC_TRTC  RTCType = "TRTC"
	RTC_AGORA RTCType = "Agora"
)

// RoomType represents the room mode/type
type RoomType string

const (
	RoomLive  RoomType = "Live"
	RoomAudio RoomType = "Audio"
)

// Room is the structure returned to clients with room information
type Room struct {
	RoomID      string   `json:"room_id"`
	OwnerUserID string   `json:"owner_userid"`
	CreateTime  int64    `json:"create_time"`
	RoomType    RoomType `json:"room_type"`
	RTCType     RTCType  `json:"rtc_type"`
}

type CreateRoomRequest struct {
	UserID   string `json:"userid"`
	RoomType string `json:"room_type"` // Live or Audio
}

type DestroyRoomRequest struct {
	RoomID string `json:"room_id"`
}

var (
	rooms   = make(map[string]Room)
	roomsMu sync.RWMutex
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// chooseRTC selects an RTC type according to a 6:4 distribution
// Rule: 0-5 => TRTC
//       6-9 => Agora
func chooseRTC() RTCType {
	// generate a random integer 0-9
	u := rand.Intn(10)
	// use switch on the random number: 0-5 => TRTC, 6-9 => Agora
	switch u {
	case 0, 1, 2, 3, 4, 5:
		return RTC_TRTC
	case 6, 7, 8, 9:
		return RTC_AGORA
	default:
		return RTC_TRTC
	}
}

func generateRoomID(userid string) string {
	return "rm_" + userid
}

// createRoomHandler handles room creation requests
func createRoomHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	rt := RoomLive
	if strings.ToLower(req.RoomType) == "audio" {
		rt = RoomAudio
	}

	id := generateRoomID(req.UserID)
	rtc := chooseRTC()

	room := Room{
		RoomID:      id,
		OwnerUserID: req.UserID,
		CreateTime:  time.Now().Unix(),
		RoomType:    rt,
		RTCType:     rtc,
	}

	roomsMu.Lock()
	rooms[id] = room
	roomsMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room)
}

// destroyRoomHandler handles room destruction requests
func destroyRoomHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req DestroyRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	roomsMu.Lock()
	defer roomsMu.Unlock()
	if _, ok := rooms[req.RoomID]; !ok {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}
	delete(rooms, req.RoomID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"result": "ok"})
}

// listRoomsHandler lists all rooms (for debugging)
func listRoomsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	roomsMu.RLock()
	defer roomsMu.RUnlock()
	list := make([]Room, 0, len(rooms))

	for _, v := range rooms {

		list = append(list, v)
	}

	// sort by CreateTime descending
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreateTime > list[j].CreateTime
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/room/create", createRoomHandler)
	mux.HandleFunc("/room/destroy", destroyRoomHandler)
	mux.HandleFunc("/room/list", listRoomsHandler)

	srv := &http.Server{
		Addr:    ":8376",
		Handler: mux,
	}

	log.Printf("server listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
