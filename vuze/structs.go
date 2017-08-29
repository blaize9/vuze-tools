package vuze

type RecoveredTorrent struct {
	Filename       string
	OrigFilepath   string
	BackupFilepath string
	Err            error
}

type RecoveredHash struct {
	Filename       string
	OrigFilepath   string
	BackupFilepath string
	Err            error
}

type FoundTorrent struct {
	Filepath string
	Err      error
}

type TorrentPathHash struct {
	Filepath string
	Hash     []uint8
	Found    bool
	Valid    bool
}

type VuzeDat struct {
	IsDatValid    bool
	HasAZ         bool
	IsAZValid     bool
	HasBak        bool
	IsBakValid    bool
	HasSaving     bool
	IsSavingValid bool
}
