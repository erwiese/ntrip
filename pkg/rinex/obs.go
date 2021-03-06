package rinex

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mholt/archiver/v3"

	"github.com/de-bkg/gognss/pkg/gnss"
)

// Options for global settings.
type Options struct {
	SatSys string // satellite systems GRE...
}

// DiffOptions sets options for file comparison.
type DiffOptions struct {
	SatSys      string // satellite systems GRE...
	CheckHeader bool   // also compare the RINEX header
}

// Coord defines a XYZ coordinate.
type Coord struct {
	X, Y, Z float64
}

// CoordNEU defines a North-, East-, Up-coordinate or eccentrity
type CoordNEU struct {
	N, E, Up float64
}

// Obs specifies a RINEX observation.
type Obs struct {
	Val float64
	LLI int8 // loss of lock indicator
	SNR int8 // signal-to-noise ratio
}

// PRN specifies a GNSS satellite.
type PRN struct {
	Sys gnss.System
	Num int8 // number
	// flags
}

func newPRN(sys gnss.System, num int8) (PRN, error) {
	/* 	if !(sys == 'G' || sys == 'R' || sys == 'E' || sys == 'J' || sys == 'C' || sys == 'I' || sys == 'S') {
		return PRN{}, fmt.Errorf("Invalid satellite system: '%v'", sys)
	} */
	if num < 1 || num > 60 {
		return PRN{}, fmt.Errorf("Check satellite number '%v%d'", sys, num)
	}
	return PRN{Sys: sys, Num: num}, nil
}

// String is a PRN Stringer.
func (prn PRN) String() string {
	return fmt.Sprintf("%s%02d", prn.Sys.Abbr(), prn.Num)
}

// SatObs conatins all observations for a satellite for a epoch.
type SatObs struct {
	Prn  PRN
	Obss map[string]Obs // L1C: obs
}

// SyncEpochs contains two epochs from different files with the same timestamp.
type SyncEpochs struct {
	Epo1 *Epoch
	Epo2 *Epoch
}

// Epoch contains a RINEX data epoch.
type Epoch struct {
	Time    time.Time // epoch time
	Flag    int8
	NumSat  uint8
	ObsList []SatObs
	//Error   error // e.g. parsing error
}

// Print pretty prints the epoch.
func (epo *Epoch) Print() {
	//fmt.Printf("%+v\n", epo)
	fmt.Printf("%s Flag: %d #prn: %d\n", epo.Time.Format(time.RFC3339Nano), epo.Flag, epo.NumSat)
	for _, satObs := range epo.ObsList {
		fmt.Printf("%v -------------------------------------\n", satObs.Prn)
		for typ, obs := range satObs.Obss {
			fmt.Printf("%s: %+v\n", typ, obs)
		}
	}
}

// PrintTab prints the epoch in a tabular format.
func (epo *Epoch) PrintTab(opts Options) {
	for _, obsPerSat := range epo.ObsList {
		printSys := false
		for _, useSys := range opts.SatSys {
			if obsPerSat.Prn.Sys.Abbr() == string(useSys) {
				printSys = true
				break
			}
		}

		if !printSys {
			continue
		}

		fmt.Printf("%s %v ", epo.Time.Format(time.RFC3339Nano), obsPerSat.Prn)
		for _, obs := range obsPerSat.Obss {
			fmt.Printf("%14.03f ", obs.Val)
		}
		fmt.Printf("\n")
	}
}

// ObsStat stores observation statistics.
type ObsStat struct {
	NumEpochs      int       `json:"numEpochs"`
	Sampling       int       `json:"sampling"`
	TimeOfFirstObs time.Time `json:"timeOfFirstObs"`
	TimeOfLastObs  time.Time `json:"timeOfLastObs"`
}

// A ObsHeader provides the RINEX Observation Header information.
type ObsHeader struct {
	RINEXVersion float32     // RINEX Format version
	RINEXType    string      // RINEX File type. O for Obs
	SatSystem    gnss.System // Satellite System. System is "Mixed" if more than one.

	Pgm   string // name of program creating this file
	RunBy string // name of agency creating this file
	Date  string // date and time of file creation TODO time.Time

	Comments []string // * comment lines

	MarkerName, MarkerNumber, MarkerType string // antennas' marker name, *number and type

	Observer, Agency string

	ReceiverNumber, ReceiverType, ReceiverVersion string
	AntennaNumber, AntennaType                    string

	Position     Coord    // Geocentric approximate marker position [m]
	AntennaDelta CoordNEU // North,East,Up deltas in [m]

	ObsTypes map[gnss.System][]string

	SignalStrengthUnit string
	Interval           float64 // Observation interval in seconds
	TimeOfFirstObs     time.Time
	TimeOfLastObs      time.Time
	LeapSeconds        int // The current number of leap seconds
	NSatellites        int // Number of satellites, for which observations are stored in the file

	labels   []string // all Header Labels found
	warnings []string
}

// ObsDecoder reads and decodes header and data records from a RINEX Obs input stream.
type ObsDecoder struct {
	// The Header is valid after NewObsDecoder or Reader.Reset. The header must exist,
	// otherwise ErrNoHeader will be returned.
	Header ObsHeader
	//b       *bufio.Reader // remove!!!
	sc      *bufio.Scanner
	epo     *Epoch // the current epoch
	syncEpo *Epoch // the snchronized epoch from a second decoder
	lineNum int
	err     error
}

// NewObsDecoder creates a new decoder for RINEX Observation data.
// The RINEX header will be read implicitly. The header must exist.
//
// It is the caller's responsibility to call Close on the underlying reader when done!
func NewObsDecoder(r io.Reader) (*ObsDecoder, error) {
	dec := &ObsDecoder{sc: bufio.NewScanner(r)}
	dec.Header, dec.err = dec.readHeader()
	return dec, dec.err
}

// Err returns the first non-EOF error that was encountered by the decoder.
func (dec *ObsDecoder) Err() error {
	if dec.err == io.EOF {
		return nil
	}

	return dec.err
}

// readHeader reads a RINEX Navigation header. If the Header does not exist,
// a ErrNoHeader error will be returned.
func (dec *ObsDecoder) readHeader() (hdr ObsHeader, err error) {
	hdr.ObsTypes = map[gnss.System][]string{}
	maxLines := 800
	rememberMe := ""
read:
	for dec.sc.Scan() {
		dec.lineNum++
		line := dec.sc.Text()
		//fmt.Print(line)

		if dec.lineNum > maxLines {
			return hdr, fmt.Errorf("Reading header failed: line %d reached without finding end of header", maxLines)
		}
		if len(line) < 60 {
			continue
		}

		// RINEX files are ASCII, so we can write:
		val := line[:60]
		key := strings.TrimSpace(line[60:])

		hdr.labels = append(hdr.labels, key)

		switch key {
		//if strings.EqualFold(key, "RINEX VERSION / TYPE") {
		case "RINEX VERSION / TYPE":
			if f64, err := strconv.ParseFloat(strings.TrimSpace(val[:20]), 32); err == nil {
				hdr.RINEXVersion = float32(f64)
			} else {
				return hdr, fmt.Errorf("parsing RINEX VERSION: %v", err)
			}
			hdr.RINEXType = strings.TrimSpace(val[20:21])
			if sys, ok := sysPerAbbr[strings.TrimSpace(val[40:41])]; ok {
				hdr.SatSystem = sys
			} else {
				err = fmt.Errorf("read header: invalid satellite system in line %d: %s", dec.lineNum, line)
				return
			}
		case "PGM / RUN BY / DATE":
			hdr.Pgm = strings.TrimSpace(val[:20])
			hdr.RunBy = strings.TrimSpace(val[20:40])
			hdr.Date = strings.TrimSpace(val[40:])
		case "COMMENT":
			hdr.Comments = append(hdr.Comments, strings.TrimSpace(val))
		case "MARKER NAME":
			hdr.MarkerName = strings.TrimSpace(val)
		case "MARKER NUMBER":
			hdr.MarkerNumber = strings.TrimSpace(val[:20])
		case "MARKER TYPE":
			hdr.MarkerType = strings.TrimSpace(val[20:40])
		case "OBSERVER / AGENCY":
			hdr.Observer = strings.TrimSpace(val[:20])
			hdr.Agency = strings.TrimSpace(val[20:])
		case "REC # / TYPE / VERS":
			hdr.ReceiverNumber = strings.TrimSpace(val[:20])
			hdr.ReceiverType = strings.TrimSpace(val[20:40])
			hdr.ReceiverVersion = strings.TrimSpace(val[40:])
		case "ANT # / TYPE":
			hdr.AntennaNumber = strings.TrimSpace(val[:20])
			hdr.AntennaType = strings.TrimSpace(val[20:40])
		case "APPROX POSITION XYZ":
			pos := strings.Fields(val)
			if len(pos) != 3 {
				return hdr, fmt.Errorf("parsing approx. position from line: %s", line)
			}
			if f64, err := strconv.ParseFloat(pos[0], 64); err == nil {
				hdr.Position.X = f64
			}
			if f64, err := strconv.ParseFloat(pos[1], 64); err == nil {
				hdr.Position.Y = f64
			}
			if f64, err := strconv.ParseFloat(pos[2], 64); err == nil {
				hdr.Position.Z = f64
			}
		case "ANTENNA: DELTA H/E/N":
			ecc := strings.Fields(val)
			if len(ecc) != 3 {
				return hdr, fmt.Errorf("parsing antenna deltas from line: %s", line)
			}
			if f64, err := strconv.ParseFloat(ecc[0], 64); err == nil {
				hdr.AntennaDelta.Up = f64
			}
			if f64, err := strconv.ParseFloat(ecc[1], 64); err == nil {
				hdr.AntennaDelta.E = f64
			}
			if f64, err := strconv.ParseFloat(ecc[2], 64); err == nil {
				hdr.AntennaDelta.N = f64
			}
		case "SYS / # / OBS TYPES":
			sysStr := val[:1]
			if sysStr == " " { // line continued
				sysStr = rememberMe
			} else {
				rememberMe = sysStr
			}

			sys, ok := sysPerAbbr[sysStr]
			if !ok {
				err = fmt.Errorf("invalid satellite system: %q: line %d", val[:1], dec.lineNum)
				return
			}

			if strings.TrimSpace(val[3:6]) != "" { // number of obstypes
				hdr.ObsTypes[sys] = strings.Fields(val[7:])
			} else {
				hdr.ObsTypes[sys] = append(hdr.ObsTypes[sys], strings.Fields(val[7:])...)
			}
		case "SIGNAL STRENGTH UNIT":
			hdr.SignalStrengthUnit = strings.TrimSpace(val[:20])
		case "INTERVAL":
			if f64, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
				hdr.Interval = f64
			}
		case "TIME OF FIRST OBS":
			t, err := time.Parse(epochTimeFormat, strings.TrimSpace(val[:43]))
			if err != nil {
				return hdr, fmt.Errorf("parsing %q: %v", key, err)
			}
			hdr.TimeOfFirstObs = t
		case "TIME OF LAST OBS":
			t, err := time.Parse(epochTimeFormat, strings.TrimSpace(val[:43]))
			if err != nil {
				return hdr, fmt.Errorf("parsing %q: %v", key, err)
			}
			hdr.TimeOfLastObs = t
		case "LEAP SECONDS": // not complete! TODO: extend
			i, err := strconv.Atoi(strings.TrimSpace(val[:6]))
			if err != nil {
				return hdr, fmt.Errorf("parsing %q: %v", key, err)
			}
			hdr.LeapSeconds = i
		case "# OF SATELLITES":
			i, err := strconv.Atoi(strings.TrimSpace(val[:6]))
			if err != nil {
				return hdr, fmt.Errorf("parsing %q: %v", key, err)
			}
			hdr.NSatellites = i
		case "END OF HEADER":
			break read
		default:
			fmt.Printf("Header field %q not handled yet\n", key)
		}
	}

	err = dec.sc.Err()
	return
}

// NextEpoch reads the observations for the next epoch.
// It returns false when the scan stops, either by reaching the end of the input or an error.
// TODO: add phase shifts
func (dec *ObsDecoder) NextEpoch() bool {
	for dec.sc.Scan() {
		dec.lineNum++
		line := dec.sc.Text()

		if len(line) < 1 {
			continue
		}

		if !strings.HasPrefix(line, "> ") {
			fmt.Printf("stream does not start with epoch line: %q\n", line) // must not be an error
			continue
		}

		//> 2018 11 06 19 00  0.0000000  0 31
		epTime, err := time.Parse(epochTimeFormat, line[2:29])
		if err != nil {
			dec.setErr(fmt.Errorf("error in line %d: %v", dec.lineNum, err))
			return false
		}

		epochFlag, err := strconv.Atoi(line[31:32])
		if err != nil {
			dec.setErr(fmt.Errorf("parsing epoch flag in line %d: %q", dec.lineNum, line))
			return false
		}

		numSat, err := strconv.Atoi(strings.TrimSpace(line[32:35]))
		if err != nil {
			dec.setErr(fmt.Errorf("error in line %d: %v", dec.lineNum, err))
			return false
		}

		//fmt.Printf("epoch: %s\n", epTime.Format(time.RFC3339Nano))
		// TODO wrap errors Go 1.13
		dec.epo = &Epoch{Time: epTime, Flag: int8(epochFlag), NumSat: uint8(numSat),
			ObsList: make([]SatObs, 0, numSat)}

		for ii := 1; ii <= numSat; ii++ {
			dec.sc.Scan()
			dec.lineNum++
			if err := dec.sc.Err(); err != nil {
				dec.setErr(fmt.Errorf("error in line %d: %v", dec.lineNum, err))
				return false
			}
			line = dec.sc.Text()

			// Parse obs line
			// fmt.Sscanf(" 1234567 ", "%5s%d", &s, &i)
			// fmt.Scanf is pretty slow in Go!? https://github.com/golang/go/issues/12275#issuecomment-133796990

			// sysHlp := []rune(line[:1])
			// sys := sysHlp[0]
			sys, ok := sysPerAbbr[line[:1]]
			if !ok {
				dec.setErr(fmt.Errorf("invalid satellite system: %q: line %d", line[:1], dec.lineNum))
				return false
			}

			snum, err := strconv.Atoi(line[1:3])
			if err != nil {
				dec.setErr(fmt.Errorf("parsing sat num in line %d: %q: %v", dec.lineNum, line, err))
				return false
			}
			prn, err := newPRN(sys, int8(snum))
			if err != nil {
				dec.setErr(fmt.Errorf("parsing sat num in line %d: %q: %v", dec.lineNum, line, err))
				return false
			}

			if strings.TrimSpace(line[3:]) == "" { // ??
				continue
			}

			obsPerTyp := make(map[string]Obs, 30) // cap
			col := 3                              // line column
			for _, typ := range dec.Header.ObsTypes[sys] {
				var val float64
				if col+14 > len(line) {
					// error ??
					dec.setErr(fmt.Errorf("obstype %s out of range in line %d: %q", typ, dec.lineNum, line))
					//break
					return false
				}

				//fmt.Printf("%q\n", line[col:col+14])
				obsStr := strings.TrimSpace(line[col : col+14])
				if obsStr != "" {
					val, err = strconv.ParseFloat(obsStr, 64)
					if err != nil {
						dec.setErr(fmt.Errorf("parsing the %s observation in line %d: %q", typ, dec.lineNum, line))
						return false
					}
				}
				col += 14

				// LLI
				if col+1 > len(line) {
					obsPerTyp[typ] = Obs{Val: val}
					break
				}
				col++
				lli, err := parseFlag(line[col-1 : col])
				if err != nil {
					dec.setErr(fmt.Errorf("parsing the %s LLI in line %d: %q: %v", typ, dec.lineNum, line, err))
					return false
				}

				// SNR
				if col+1 > len(line) {
					obsPerTyp[typ] = Obs{Val: val, LLI: int8(lli)}
					break
				}
				col++
				snr, err := parseFlag(line[col-1 : col])
				if err != nil {
					dec.setErr(fmt.Errorf("parsing the %s SNR in line %d: %q: %v", typ, dec.lineNum, line, err))
					return false
				}

				obsPerTyp[typ] = Obs{Val: val, LLI: int8(lli), SNR: int8(snr)}
			}
			dec.epo.ObsList = append(dec.epo.ObsList, SatObs{Prn: prn, Obss: obsPerTyp})
		}
		return true
	}

	if err := dec.sc.Err(); err != nil {
		dec.setErr(fmt.Errorf("read epoch scanner error: %v", err))
	}

	return false // EOF
}

// Epoch returns the most recent epoch generated by a call to NextEpoch.
func (dec *ObsDecoder) Epoch() *Epoch {
	return dec.epo
}

// SyncEpoch returns the current pair of time-synchronized epochs from two RINEX Obs input streams.
func (dec *ObsDecoder) SyncEpoch() SyncEpochs {
	return SyncEpochs{dec.epo, dec.syncEpo}
}

// setErr records the first error encountered.
func (dec *ObsDecoder) setErr(err error) {
	if dec.err == nil || dec.err == io.EOF {
		dec.err = err
	}
}

// sync returns a stream of time-synchronized epochs from two RINEX Obs input streams.
func (dec *ObsDecoder) sync(dec2 *ObsDecoder) bool {
	var epoF1, epoF2 *Epoch
	for dec.NextEpoch() {
		epoF1 = dec.Epoch()
		//fmt.Printf("%s: got f1\n", epoF1.Time)

		if epoF2 != nil {
			if epoF1.Time.Equal(epoF2.Time) {
				dec.syncEpo = epoF2
				return true
			} else if epoF2.Time.After(epoF1.Time) {
				continue // next epo1 needed
			}
		}

		// now we need the next epo2
		for dec2.NextEpoch() {
			epoF2 = dec2.Epoch()
			//fmt.Printf("%s: got f2\n", epoF2.Time)
			if epoF2.Time.Equal(epoF1.Time) {
				dec.syncEpo = epoF2
				return true
			} else if epoF2.Time.After(epoF1.Time) {
				break // next epo1 needed
			}
		}
	}

	if err := dec2.Err(); err != nil {
		dec.setErr(fmt.Errorf("stream2 decoder error: %v", err))
	}
	return false
}

// ObsFile contains fields and methods for RINEX observation files.
// Use NewObsFil() to instantiate a new ObsFile.
type ObsFile struct {
	*RnxFil
	Header ObsHeader
	Opts   Options
}

// NewObsFile returns a new ObsFile.
func NewObsFile(filepath string) (*ObsFile, error) {
	// must file exist?
	obsFil := &ObsFile{RnxFil: &RnxFil{Path: filepath}, Opts: Options{}}
	err := obsFil.parseFilename()
	return obsFil, err
}

// Diff compares two RINEX obs files.
func (f *ObsFile) Diff(obsFil2 *ObsFile) error {
	// file 1
	r, err := os.Open(f.Path)
	if err != nil {
		return fmt.Errorf("open obs file: %v", err)
	}
	defer r.Close()
	dec, err := NewObsDecoder(r)
	if err != nil {
		return err
	}

	// file 2
	r2, err := os.Open(obsFil2.Path)
	if err != nil {
		return fmt.Errorf("open obs file: %v", err)
	}
	defer r2.Close()
	dec2, err := NewObsDecoder(r2)
	if err != nil {
		return err
	}

	nSyncEpochs := 0
	for dec.sync(dec2) {
		nSyncEpochs++
		syncEpo := dec.SyncEpoch()

		diff := diffEpo(syncEpo, f.Opts)
		if diff != "" {
			fmt.Printf("diff: %s\n", diff)
		}
	}
	if err := dec.Err(); err != nil {
		return fmt.Errorf("read epochs error: %v", err)
	}

	return nil
}

// Stat gathers some observation statistics.
func (f *ObsFile) Stat() (stat ObsStat, err error) {
	r, err := os.Open(f.Path)
	if err != nil {
		return
	}
	defer r.Close()
	dec, err := NewObsDecoder(r)
	if err != nil {
		return
	}

	numOfEpochs := 0
	intervals := make([]time.Duration, 0, 10)
	var epo, epoPrev *Epoch

	for dec.NextEpoch() {
		numOfEpochs++
		epo = dec.Epoch()
		if numOfEpochs == 1 {
			stat.TimeOfFirstObs = epo.Time
		}

		if epoPrev != nil && len(intervals) <= 10 {
			intervals = append(intervals, epo.Time.Sub(epoPrev.Time))
		}

		epoPrev = epo
	}
	if err = dec.Err(); err != nil {
		return
	}

	stat.TimeOfLastObs = epoPrev.Time
	stat.NumEpochs = numOfEpochs

	// check sampling rate
	// for _, dur := range intervals {
	// 	dur.Seconds()
	// }

	// sampling
	// LLIs

	return
}

// Compress an observation file using Hatanaka first and then gzip.
// The source file will be removed if the compression finishes without errors.
func (f *ObsFile) Compress() error {
	if f.Format == "crx" && f.Compression == "gz" {
		return nil
	}
	if f.Format == "rnx" && f.Compression != "" {
		return fmt.Errorf("compressed file is not Hatanaka compressed: %s", f.Path)
	}

	err := f.Rnx2crx()
	if err != nil {
		return err
	}

	err = archiver.CompressFile(f.Path, f.Path+".gz")
	if err != nil {
		return err
	}
	os.Remove(f.Path)
	f.Path = f.Path + ".gz"
	f.Compression = "gz"

	return nil
}

// IsHatanakaCompressed returns true if the obs file is Hatanaka compressed, otherwise false.
func (f *ObsFile) IsHatanakaCompressed() bool {
	if f.Format == "crx" {
		return true
	}

	return false
}

// Rnx2crx creates a copy of the file that is Hatanaka-compressed (compact RINEX).
// Rnx2crx returns the filepath of the compressed file.
// see http://terras.gsi.go.jp/ja/crx2rnx.html
func (f *ObsFile) Rnx2crx() error {
	rnxFilePath := f.Path

	if f.IsHatanakaCompressed() {
		return nil
	}

	tool, err := exec.LookPath("RNX2CRX")
	if err != nil {
		return err
	}

	dir, rnxFil := filepath.Split(rnxFilePath)

	// Build name of target file
	crxFil := ""
	if Rnx2FileNamePattern.MatchString(rnxFil) {
		crxFil = Rnx2FileNamePattern.ReplaceAllString(rnxFil, "${2}${3}${4}${5}.${6}d")
	} else if Rnx3FileNamePattern.MatchString(rnxFil) {
		crxFil = Rnx3FileNamePattern.ReplaceAllString(rnxFil, "${2}.crx")
	} else {
		return fmt.Errorf("file %s with no standard RINEX extension", rnxFil)
	}

	//fmt.Printf("rnxFil: %s - crxFil: %s\n", rnxFil, crxFil)

	if crxFil == "" || rnxFil == crxFil {
		return fmt.Errorf("Could not build compressed filename for %s", rnxFil)
	}

	// Run compression tool
	cmd := exec.Command(tool, rnxFilePath, "-d", "-f")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("cmd %s failed: %v: %s", tool, err, stderr.Bytes())
	}

	// Return filepath
	crxFilePath := filepath.Join(dir, crxFil)
	if _, err := os.Stat(crxFilePath); os.IsNotExist(err) {
		return fmt.Errorf("compressed file does not exist: %s", crxFilePath)
	}

	f.Path = crxFilePath
	f.Format = "crx"

	return nil
}

// Crx2rnx decompresses a Hatanaka-compressed file. Crx2rnx returns the filepath of the decompressed file.
// see http://terras.gsi.go.jp/ja/crx2rnx.html
func (f *ObsFile) Crx2rnx() error {
	crxFilepath := f.Path

	if !f.IsHatanakaCompressed() {
		fmt.Printf("File is already Hatanaka uncompressed\n")
		return nil
	}

	tool, err := exec.LookPath("CRX2RNX")
	if err != nil {
		return err
	}

	dir, crxFil := filepath.Split(crxFilepath)

	// Build name of target file
	rnxFil := ""
	if Rnx2FileNamePattern.MatchString(crxFil) {
		rnxFil = Rnx2FileNamePattern.ReplaceAllString(crxFil, "${2}${3}${4}${5}.${6}o")
	} else if Rnx3FileNamePattern.MatchString(crxFil) {
		rnxFil = Rnx3FileNamePattern.ReplaceAllString(crxFil, "${2}.rnx")
	} else {
		return fmt.Errorf("file %s with no standard RINEX extension", crxFil)
	}

	if rnxFil == "" || rnxFil == crxFil {
		return fmt.Errorf("Could not build uncompressed filename for %s", crxFil)
	}

	// Run compression tool
	cmd := exec.Command(tool, crxFilepath, "-d", "-f")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("cmd %s failed: %v: %s", tool, err, stderr.Bytes())
	}

	// Return filepath
	rnxFilePath := filepath.Join(dir, rnxFil)
	if _, err := os.Stat(rnxFilePath); os.IsNotExist(err) {
		return fmt.Errorf("compressed file does not exist: %s", rnxFilePath)
	}

	f.Path = rnxFilePath
	return nil
}

// Rnx3Filename returns the filename following the RINEX3 convention.
// In most cases we must read the read the header. The countrycode must come from an external source.
// DO NOT USE! Must parse header first!
func (f *ObsFile) Rnx3Filename() (string, error) {
	if f.DataFreq == "" || f.FilePeriod == "" {
		r, err := os.Open(f.Path)
		if err != nil {
			return "", err
		}
		defer r.Close()
		dec, err := NewObsDecoder(r)
		if err != nil {
			return "", err
		}

		if dec.Header.Interval != 0 {
			f.DataFreq = fmt.Sprintf("%02d%s", int(dec.Header.Interval), "S")
		}

		f.DataType = fmt.Sprintf("%s%s", dec.Header.SatSystem.Abbr(), "O")
	}

	// Station Identifier
	if len(f.FourCharID) != 4 {
		return "", fmt.Errorf("FourCharID: %s", f.FourCharID)
	}

	if len(f.CountryCode) != 3 {
		return "", fmt.Errorf("CountryCode: %s", f.CountryCode)
	}

	var fn strings.Builder
	fn.WriteString(f.FourCharID)
	fn.WriteString(strconv.Itoa(f.MonumentNumber))
	fn.WriteString(strconv.Itoa(f.ReceiverNumber))
	fn.WriteString(f.CountryCode)

	fn.WriteString("_")

	if f.DataSource == "" {
		fn.WriteString("U")
	} else {
		fn.WriteString(f.DataSource)
	}

	fn.WriteString("_")

	// StartTime
	//BRUX00BEL_R_20183101900_01H_30S_MO.rnx
	fn.WriteString(strconv.Itoa(f.StartTime.Year()))
	fn.WriteString(fmt.Sprintf("%03d", f.StartTime.YearDay()))
	fn.WriteString(fmt.Sprintf("%02d", f.StartTime.Hour()))
	fn.WriteString(fmt.Sprintf("%02d", f.StartTime.Minute()))
	fn.WriteString("_")

	fn.WriteString(f.FilePeriod)
	fn.WriteString("_")

	fn.WriteString(f.DataFreq)
	fn.WriteString("_")

	fn.WriteString(f.DataType)

	if f.Format == "crx" {
		fn.WriteString(".crx")
	} else {
		fn.WriteString(".rnx")
	}

	if len(fn.String()) != 38 {
		return "", fmt.Errorf("invalid filename: %s", fn.String())
	}

	// Rnx3 Filename: total: 41-42 obs, 37-38 eph.

	return fn.String(), nil
}

func parseFlag(str string) (int, error) {
	if str == " " {
		return 0, nil
	}

	return strconv.Atoi(str)
}

// get decimal part of a float.
func getDecimal(f float64) float64 {
	// or big.NewFloat(f).Text("f", 6)
	fBig := big.NewFloat(f)
	fint, _ := fBig.Int(nil)
	intf := new(big.Float).SetInt(fint)
	//fmt.Printf("accuracy: %d\n", acc)
	resBig := new(big.Float).Sub(fBig, intf)
	ff, _ := resBig.Float64()
	return ff
}

// compare two epochs
func diffEpo(epochs SyncEpochs, opts Options) string {
	epo1, epo2 := epochs.Epo1, epochs.Epo2
	epoTime := epo1.Time
	// if epo1.NumSat != epo2.NumSat {
	// 	return fmt.Sprintf("epo %s: different number of satellites: fil1: %d fil2:%d", epoTime, epo1.NumSat, epo2.NumSat)
	// }

	for _, obs := range epo1.ObsList {
		printSys := false
		for _, useSys := range opts.SatSys {
			if obs.Prn.Sys.Abbr() == string(useSys) {
				printSys = true
				break
			}
		}

		if !printSys {
			continue
		}

		obs2, err := getObsByPRN(epo2.ObsList, obs.Prn)
		if err != nil {
			fmt.Printf("%v\n", err)
			continue
		}

		diffObs(obs, obs2, epoTime, obs.Prn)
	}

	return ""
}

func getObsByPRN(obslist []SatObs, prn PRN) (SatObs, error) {
	for _, obs := range obslist {
		if obs.Prn == prn {
			return obs, nil
		}
	}

	return SatObs{}, fmt.Errorf("No oberservations found for prn %v", prn)
}

func diffObs(obs1, obs2 SatObs, epoTime time.Time, prn PRN) string {
	deltaPhase := 0.005
	checkSNR := false
	for k, o1 := range obs1.Obss {
		if o2, ok := obs2.Obss[k]; ok {
			val1, val2 := o1.Val, o2.Val
			if strings.HasPrefix(k, "L") { // phase observations
				val1 = getDecimal(val1)
				val2 = getDecimal(val2)
			}
			if (o1.LLI != o2.LLI) || (math.Abs(val1-val2) > deltaPhase) {
				fmt.Printf("%s %v %02d %s %s %14.03f %d %d | %14.03f %d %d\n", epoTime.Format(time.RFC3339Nano), prn.Sys, prn.Num, k[:1], k, val1, o1.LLI, o1.SNR, val2, o2.LLI, o2.SNR)
			} else if checkSNR && o1.SNR != o2.SNR {
				fmt.Printf("%s %v %02d %s %s %14.03f %d %d | %14.03f %d %d\n", epoTime.Format(time.RFC3339Nano), prn.Sys, prn.Num, k[:1], k, val1, o1.LLI, o1.SNR, val2, o2.LLI, o2.SNR)
			}

			// if o1.SNR != o2.SNR {
			// 	fmt.Printf("%s: SNR: %s: %d %d\n", epoTime.Format(time.RFC3339Nano), k, o1.SNR, o2.SNR)
			// }
			// if val1 != val2 {
			// 	fmt.Printf("%s: val: %s: %14.03f %14.03f\n", epoTime.Format(time.RFC3339Nano), k, val1, val2)
			// }
		} else {
			fmt.Printf("Key %q does not exist\n", k)
		}

	}

	return ""
}
