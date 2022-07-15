package main

import (
	"flag"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/JoshuaDoes/logger"
	"github.com/StickFightDev/steamcmd"
	"github.com/microcosm-cc/bluemonday"
)

//Command-line flags and their defaults
var (
	//SteamCMD login
	steamKey      = ""
	steamUsername = "anonymous"
	steamPassword = ""
	steamCmdDir   = ""

	//Server config
	address       = "0.0.0.0:1337"
	maxBufferSize = 8192
	maxLobbies    = 100

	//Logging
	verbosityLevel  = 0
	logPlayerUpdate = false
)

//The server itself
var (
	log        *logger.Logger     //Console logger
	scmd       *steamcmd.SteamCmd //SteamCMD
	server     *Server            //StickFightDev server
	randomizer *rand.Rand         //Seed for random numbers

	stripTags *bluemonday.Policy
)

func init() {
	flag.StringVar(&steamKey, "steamKey", steamKey, "The API key to use when asking Steam for usernames")
	flag.StringVar(&steamUsername, "username", steamUsername, "The username for the Steam account that owns Stick Fight")
	flag.StringVar(&steamPassword, "password", steamPassword, "The password for the Steam account that owns Stick Fight")
	flag.StringVar(&steamCmdDir, "steamCmdDir", steamCmdDir, "The directory holding the root of your SteamCmd install")
	flag.StringVar(&address, "address", address, "The IP and port to serve on")
	flag.IntVar(&maxBufferSize, "maxBufferSize", maxBufferSize, "The maximum buffer size of expected incoming packets")
	flag.IntVar(&maxLobbies, "maxLobbies", maxLobbies, "The maximum amount of lobbies to allow")
	flag.IntVar(&verbosityLevel, "verbosity", verbosityLevel, "The verbosity level of debug log output")
	flag.BoolVar(&logPlayerUpdate, "logPlayerUpdate", logPlayerUpdate, "Enables logging playerUpdate packets")
	flag.Parse()

	log = logger.NewLogger("sf:srv", verbosityLevel)
	//stripTags = bluemonday.StripTagsPolicy()
	stripTags = bluemonday.StrictPolicy()
}

func main() {
	var err error

	//Initialize steamcmd
	log.Info("Logging into Steam...")
	scmd = steamcmd.New(steamUsername, steamPassword)
	if verbosityLevel == 2 {
		scmd.Debug = true
	}
	if err := scmd.EnsureInstalled(); err != nil {
		log.Fatal(err)
	}
	if err := scmd.CheckLogin(); err != nil {
		log.Fatal(err)
	}

	log.Trace("Seeding randomizer...")
	randomizer = rand.New(rand.NewSource(time.Now().UnixNano()))

	log.Trace("Loading default levels...")
	os.Mkdir("maps", 0755)

		for i := int32(1); i <= 124; i++ {
			if i == 0 || i == 102 {
				continue
			}
			defaultLevels = append(defaultLevels, newLevelLandfall(i))
		}

	lobbyLevels, err = LoadWorkshopMaps(
		2362135194, 2362150591, 2362151526, 2362151645,
		2362151790, 2362151892, 2362152017, 2362152135,
	)
	if err != nil {
		log.Fatal(err)
	}
/*	defaultLevels, err = LoadWorkshopMaps(
		2200042304, 2200047921, 2200051799, 2200056261,
		2200058789, 2200062744, 2200069103, 2200073817,
		2200078748, 2200086047, 2200090348, 2200092415,
		2200098344, 2200100283, 2200102885, 2200106023,
		2200107893, 2200109408, 2200112035, 2200113733,
		2200116667, 2200118707, 2200119774, 2200122075,
		2200123719, 2200126432, 2200129454, 2200131581,
		2200137191, 2200140347, 2200142489, 2200145858,
		2200147947, 2200152837, 2200157521, 2200161014,
		2200163529, 2200166476, 2200561630, 2200566979,
		2200572201, 2200577287, 2200582772, 2200585235,
		2200614142, 2200631612, 2205919112, 2205950305,
		2205969235, 2205978243, 2206006449, 2206027577,
		2206041190, 2206047592, 2206225526, 2206344259,
		2206360656, 2206388990, 2206397608, 2206407020,
		2206417499, 2206431577, 2206435705, 2206453774,
		2206460543, 2206464628, 2206467263, 2206518302,
		2206539363, 2206542211, 2208826631, 2208831597,
		2208836118, 2208843238, 2208847724, 2208851046,
		2208859746, 2208864249, 2208897753, 2208910916,
		2208914173, 2208916640, 2208919069, 2208928980,
		2208932119, 2208933857, 2208943514, 2208946630,
		2208996342, 2209006315, 2209010129, 2209020283,
		2209033059, 2209046306, 2209407522, 2209422159,
		2209838063, 2209860643, 2209878071, 2209902974,
		2209906828,
	)
	if err != nil {
		log.Error(err)
	}*/

	//Run the server
	log.Info("Starting the server...")
	server = NewServer(address)
	go server.Run()
	defer server.Close()

	log.Trace("Waiting for exit call from system")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT)
	<-sc

	log.Trace("SIGINT received!")
	log.Info("Good-bye!")
}
