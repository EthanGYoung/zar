package manager

import (
	"bufio"
	"log"
	"os"
	"strings"
)

// CManager ("configmanager") is a struct for writing image files from a configuration file
// The configuration file will specify which files to read (relative to root dir)
// and in what order to put them in img file.
type CManager struct {
        // Inherits manager's methods
        *ZarManager

        // Format of the input file (e.g. YAML, csv, ..)
        Format string

        // The configuration file with the structure of the img file specified in the Format in the Format field
        ConfigFile *os.File
}

// WalkDir implements Manager.WalkDir. Overrides zarManager'si
func (c *CManager) WalkDir(dir string, foldername string, root bool) {

        switch c.Format {
        case "seq":
        // seq Format is as follows
        // <File (f) or Start Dir (sd) or End Dir (ed) > | < path excluding file name > | < name > \n
        // TODO: Create a file reader struct to allow for generic reading of different Formats.
        // TODO: For now, prototype will always assume seq
        default:
                log.Fatalf("Config Format not recognized")
        }

        // Close file once scanning is complete
        defer c.ConfigFile.Close()
        scanner := bufio.NewScanner(c.ConfigFile)
        scanner.Split(bufio.ScanLines)

        // Read each line in the config file
        for scanner.Scan() {
                // Parse the line TODO: Save path along way so config does not need path and name separate
                s := strings.Split(scanner.Text(), "|")
                action, path, name := s[0], s[1], s[2]

                switch action {
                case "f":
                        c.IncludeFile(name, path, 0, 0)
                        // TODO: change 0 to valid timestamp
                        // TODO: change the second 0 to valid file mode
                case "sd":
                        c.IncludeFolderBegin(name, 0, 0)
                case "ed":
                        c.IncludeFolderEnd()
                default:
                        log.Fatalf("Config action not recognized")
                }
        }
}
