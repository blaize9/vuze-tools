package vuze

import (
	"errors"
	"github.com/IncSW/go-bencode"
	"github.com/KyleBanks/go-kit/log"
	"github.com/blaize9/vuze-tools/config"
	"github.com/blaize9/vuze-tools/utils"
	"github.com/djherbis/times"
	pbar "github.com/pmalek/pb"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

func CheckHashStorage() (storage HashStorage, lastMod time.Time, BackupDirCount int, UniqueHashCount int) {
	err := utils.LoadStruct(filepath.Join(config.GetAzRecoverPath(), "hashstorage.struct"), &storage)
	if err != nil {
		log.Errorf("Error reading hashstorage [%s]", err)
	}
	lastMod = storage.LastModified
	BackupDirCount = len(storage.BackupDirectories)
	UniqueHashCount = len(storage.HashMap)

	log.Infof("HashStorage was last modified %s and contains %d directories with %d unique hashes", lastMod, BackupDirCount, UniqueHashCount)
	return
}

func ReadDownloadsConfig() (interface{}, error) {
	file, er := ioutil.ReadFile(config.GetAzDownloadsConfig())
	if er != nil {
		return nil, errors.New("Unable to open vuze downloads config")
	}
	data, err := bencode.Unmarshal(file)
	if err != nil {
		return nil, errors.New("Unable to unmarshal vuze downloads config")
	}
	return data, nil
}

func SaveTorrentFromActive(ActivePath string, destFilepath string) (bool, error) {
	file, err := ioutil.ReadFile(ActivePath)
	if err != nil {
		return false, err
	}
	data, err := bencode.Unmarshal(file)
	if err != nil {
		return false, err
	}

	datam := data.(map[string]interface{})
	bencodeM := map[string]interface{}{}

	for k, v := range datam {
		if k == "comment" || k == "created by" || k == "creation date" || k == "encoding" ||
			k == "info" || k == "announce" || k == "announce-list" {
			bencodeM[k] = v
		}
	}

	m, _ := bencode.Marshal(bencodeM)

	destFile, err := os.Create(destFilepath)
	if err != nil {
		return false, err
	}
	defer destFile.Close()

	srcStat, err := times.Stat(ActivePath)
	if err != nil {
		return false, err
	}

	_, err = io.WriteString(destFile, utils.ByteToString(m))
	if err != nil {
		return false, err
	}
	err = destFile.Sync()

	var createdTime time.Time
	if srcStat.HasBirthTime() {
		createdTime = srcStat.BirthTime()
	} else {
		createdTime = srcStat.ModTime()
	}

	if err := os.Chtimes(destFilepath, createdTime, createdTime); err != nil {
		return false, err
	}

	return true, err
}

// Directories inside Path must contain ####-##-##
func GetAllVuzeBackupDirectores(Path string) (dirs []string) {
	dirmatch, _ := regexp.Compile("\\d{4}-\\d{2}-\\d{2}")
	foundDirs := utils.GetAllSubDirectories(Path)
	sort.Sort(sort.Reverse(sort.StringSlice(foundDirs)))

	for _, dir := range foundDirs {
		if dirmatch.MatchString(dir) {
			if utils.DirExists(dir) {
				dirs = append(dirs, dir)
			}
		}
	}

	return dirs
}

func ProcessActiveDirectory(activePath string) map[string]VuzeDat {
	Hashes := map[string]VuzeDat{}

	datFileCount := 0
	files, _ := ioutil.ReadDir(activePath)
	for _, finfo := range files {
		if filepath.Ext(finfo.Name()) == ".dat" {
			datFileCount++
		}
	}

	bar := pbar.StartNew(datFileCount)
	for _, finfo := range files {
		ActivePath := config.GetAzActivePath() + "/" + finfo.Name()

		hash := strings.TrimSuffix(ActivePath, filepath.Ext(ActivePath))
		baseFilename := filepath.Base(ActivePath)
		baseFilenameWithoutExt := strings.TrimSuffix(baseFilename, filepath.Ext(baseFilename))
		if filepath.Ext(ActivePath) == ".dat" {
			vuzeDat := VuzeDat{}

			AZ := hash + ".dat._AZ"
			BAK := hash + ".dat.bak"
			SAVING := hash + ".dat.saving"

			if utils.IsBencodeFileValid(ActivePath) {
				vuzeDat.IsDatValid = true
			}
			if utils.FileExists(AZ) {
				vuzeDat.HasAZ = true
				if utils.IsBencodeFileValid(AZ) {
					vuzeDat.IsAZValid = true
				}
			}
			if utils.FileExists(BAK) {
				vuzeDat.HasBak = true
				if utils.IsBencodeFileValid(BAK) {
					vuzeDat.IsBakValid = true
				}
			}
			if utils.FileExists(SAVING) {
				vuzeDat.HasSaving = true
				if utils.IsBencodeFileValid(SAVING) {
					vuzeDat.IsSavingValid = true
				}
			}

			Hashes[baseFilenameWithoutExt] = vuzeDat
			bar.Increment()
		}

	}

	bar.FinishPrint("Finished Scanning. Start the recovery!")
	return Hashes
}

func ScanDownloadsConfig() ([]TorrentPathHash, error) {
	data, err := ReadDownloadsConfig()
	if err != nil {
		log.Errorf("%v", err)
		return nil, err
	}
	datam := data.(map[string]interface{})
	torrents := []TorrentPathHash{}
	if utils.IsMap(datam) {
		for k, v := range datam {
			if k == "downloads" {
				for _, vv := range v.([]interface{}) {
					torrent := TorrentPathHash{}
					for kkk, vvv := range vv.(map[string]interface{}) {
						if kkk == "torrent" {
							//files++
							torrent_filepath := utils.ByteToString(vvv.([]uint8))
							if torrent_filepath != "" {
								torrent.Filepath = torrent_filepath
								if utils.FileExists(torrent_filepath) {
									torrent.Found = true
									if utils.IsTorrentValid(torrent_filepath) == nil {
										torrent.Valid = true
									}
								}
							}

						}

						if kkk == "torrent_hash" {
							torrent.Hash = vvv.([]uint8)
						}
					}
					torrents = append(torrents, torrent)
				}
			}
		}
		return torrents, nil
	}
	return torrents, errors.New("downloads.config is not valid!")
}

func ShuffleBackupDirectories(slice []string) {
	for i := range slice {
		j := rand.Intn(i + 1)
		slice[i], slice[j] = slice[j], slice[i]
	}
}
