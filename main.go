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

// RTCType 表示底层 RTC 服务类型
type RTCType string

const (
	RTC_TRTC  RTCType = "TRTC"
	RTC_AGORA RTCType = "Agora"
)

// RoomType 表示房间模式
type RoomType string

const (
	RoomLive  RoomType = "Live"
	RoomAudio RoomType = "Audio"
)

// Link 表示两条链路及其分配比例
type Link struct {
	Name  string  `json:"name"`
	Share float64 `json:"share"` // 比例，如 0.6 或 0.4
}

// Room 为返回给客户端的房间信息结构
type Room struct {
	RoomID      string   `json:"room_id"`
	OwnerUserID string   `json:"owner_userid"`
	CreateTime  int64    `json:"create_time"`
	RoomType    RoomType `json:"room_type"`
	RTCType     RTCType  `json:"rtc_type"`
}

type CreateRoomRequest struct {
	UserID   string `json:"userid"`
	RoomType string `json:"room_type"` // Live 或 Audio
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

// chooseRTC 根据比例来随机分配rtc房间类型
// 规则 TRTC:Agora => 6:4
// 0-5 => TRTC
// 6-9 => Agora
func chooseRTC() RTCType {
	// 生成的是 0-9 的随机数
	u := rand.Intn(10)
	// 使用 switch 对随机数进行判断：0-5 => TRTC，6-9 => Agora
	switch u {
	case 0, 1, 2, 3, 4, 5:
		return RTC_TRTC
	case 6, 7, 8, 9:
		return RTC_AGORA
	default:
		// 按要求 userid 必定为数字字符，这里作为兜底返回 TRTC
		return RTC_TRTC
	}
}

func generateRoomID(userid string) string {
	// return strconv.FormatInt(time.Now().UnixNano(), 36) + "-" + strconv.Itoa(rand.Intn(10000))
	return "rm_" + userid
}

// createRoomHandler 创建房间接口
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

// destroyRoomHandler 解散房间接口
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

// listRoomsHandler 列出所有房间（调试用）
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

	// Sort by CreateTime in descending order
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
