package character

// AnimationBank is shared by all four models. Tracks bind by canonical node name.
const AnimationBank = "animations/shared.glb"

// Model describes a model-only GLB with embedded textures and its shared rig.
type Model struct {
	Role, File        string
	Triangles, Joints int
}

// Models returns independent metadata; it does not load or embed model data.
func Models() [4]Model { return models }

var models = [...]Model{
	{"scientist", "models/scientist.glb", 1320, 66},
	{"guard", "models/guard.glb", 1456, 66},
	{"assault", "models/assault.glb", 2181, 66},
	{"policeman", "models/policeman.glb", 782, 66},
}

// Clip describes an unmodified source action and its sampling semantics.
type Clip struct {
	Alias, Source string
	Seconds       float64
	Loop          bool
}

// Clips returns the shared action table. Copies cannot change other actors.
func Clips() [20]Clip { return clips }

var clips = [...]Clip{
	{"idle", "Idle_A", 3.125, true},
	{"walk", "Walk", 1.6666666269302368, true},
	{"run", "Jog", 1.1666666269302368, true},
	{"crouch", "Crouch_Idle", 3.6666667461395264, true},
	{"crouch-walk", "Crouch_Walk", 2.5, true},
	{"crawl", "Crawl", 1.7083333730697632, true},
	{"jump-start", "Jump_Start", 1.6666666269302368, false},
	{"jump-air", "Jump_air", 3.125, true},
	{"jump-land", "Jump_Land", 1.5833333730697632, false},
	{"strafe-left", "Strafe_left", 1, true},
	{"strafe-right", "Strafe_right", 1, true},
	{"aware", "Confused", 2.9166667461395264, false},
	{"use", "Interact", 2.5, false},
	{"pistol-idle", "Pistol_Idle", 2.0833332538604736, true},
	{"pistol-fire", "Pistol_Shoot", 0.7916666865348816, false},
	{"pistol-reload", "Pistol_Reload", 2.0833332538604736, false},
	{"talk", "Sitting_Talking", 3.6666667461395264, false},
	{"death-a", "Death_A", 4.5, false},
	{"death-c", "Death_C", 2.2083332538604736, false},
	{"death-d", "Death_D", 2.1666667461395264, false},
}

// ClipID returns the zero-based action-table index, or -1 for an unknown alias.
// A renderer must resolve Source by node/clip name, not assume GLB array order.
func ClipID(alias string) int {
	for i, clip := range clips {
		if clip.Alias == alias {
			return i
		}
	}
	return -1
}

// Duration returns source seconds, or zero for an unknown alias.
func Duration(alias string) float64 {
	i := ClipID(alias)
	if i < 0 {
		return 0
	}
	return clips[i].Seconds
}

func duration(alias string) float64 { return Duration(alias) }
