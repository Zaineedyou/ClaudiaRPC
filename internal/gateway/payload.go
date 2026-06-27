package gateway

type Payload struct {
	Op int         `json:"op"`
	D  interface{} `json:"d"`
	S  *int        `json:"s,omitempty"`
	T  *string     `json:"t,omitempty"`
}

type Identify struct {
	Token      string             `json:"token"`
	Properties IdentifyProperties `json:"properties"`
	Intents    int                `json:"intents"`
}

type IdentifyProperties struct {
	OS      string `json:"os"`
	Browser string `json:"browser"`
	Device  string `json:"device"`
}

type UpdateStatus struct {
	Since      *int       `json:"since"`
	Activities []Activity `json:"activities"`
	Status     string     `json:"status"`
	AFK        bool       `json:"afk"`
}

type ActivityMetadata struct {
	ButtonURLs []string `json:"button_urls"`
}

type Activity struct {
	Name          string             `json:"name"`
	Type          int                `json:"type"`
	ApplicationID string             `json:"application_id,omitempty"`
	Details       string             `json:"details,omitempty"`
	State         string             `json:"state,omitempty"`
	Timestamps    *ActivityTimestamp `json:"timestamps,omitempty"`
	Assets        *ActivityAssets    `json:"assets,omitempty"`
	Buttons       []string           `json:"buttons,omitempty"`
	Metadata      *ActivityMetadata  `json:"metadata,omitempty"`
}

type ActivityTimestamp struct {
	Start int64 `json:"start,omitempty"`
	End   int64 `json:"end,omitempty"`
}

type ActivityAssets struct {
	LargeImage string `json:"large_image,omitempty"`
	LargeText  string `json:"large_text,omitempty"`
	SmallImage string `json:"small_image,omitempty"`
	SmallText  string `json:"small_text,omitempty"`
}

type Hello struct {
	HeartbeatInterval int `json:"heartbeat_interval"`
}

type RPCData struct {
	AppName        string `json:"app_name"`
	ClientID       string `json:"client_id"`
	Details        string `json:"details"`
	State          string `json:"state"`
	LargeImage     string `json:"large_image"`
	LargeText      string `json:"large_text"`
	SmallImage     string `json:"small_image"`
	SmallText      string `json:"small_text"`
	Button1Label   string `json:"button1_label"`
	Button1URL     string `json:"button1_url"`
	Button2Label   string `json:"button2_label"`
	Button2URL     string `json:"button2_url"`
	TimestampStart string `json:"timestamp_start"`
	TimestampEnd   string `json:"timestamp_end"`
	Type           int    `json:"type"`
}
