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
)

var (
	//Command-line flags and their defaults
	steamKey        = ""
	steamUsername   = "anonymous"
	steamPassword   = ""
	steamCmdDir     = ""
	address         = "0.0.0.0:1337"
	maxBufferSize   = 8192
	maxLobbies      = 100
	verbosityLevel  = 0
	logPlayerUpdate = false

	//Things to be used by the server
	log        *logger.Logger
	scmd       *steamcmd.SteamCmd
	server     *Server
	randomizer *rand.Rand
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
}

func main() {
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
	/*
		for i := int32(1); i <= 124; i++ {
			if i == 0 || i == 102 {
				continue
			}
			defaultLevels = append(defaultLevels, newLevelLandfall(i))
		}
	*/
	lobbyLevels = []*Level{
		newLevelCustomOnline(2362135194),
		newLevelCustomOnline(2362150591),
		newLevelCustomOnline(2362151526),
		newLevelCustomOnline(2362151645),
		newLevelCustomOnline(2362151790),
		newLevelCustomOnline(2362151892),
		newLevelCustomOnline(2362152017),
		newLevelCustomOnline(2362152135),
	}
	defaultLevels = []*Level{
		newLevelCustomOnline(2200042304),
		newLevelCustomOnline(2200047921),
		newLevelCustomOnline(2200051799),
		newLevelCustomOnline(2200056261),
		newLevelCustomOnline(2200058789),
		newLevelCustomOnline(2200062744),
		newLevelCustomOnline(2200069103),
		newLevelCustomOnline(2200073817),
		newLevelCustomOnline(2200078748),
		newLevelCustomOnline(2200086047),
		newLevelCustomOnline(2200090348),
		newLevelCustomOnline(2200092415),
		newLevelCustomOnline(2200098344),
		newLevelCustomOnline(2200100283),
		newLevelCustomOnline(2200102885),
		newLevelCustomOnline(2200106023),
		newLevelCustomOnline(2200107893),
		newLevelCustomOnline(2200109408),
		newLevelCustomOnline(2200112035),
		newLevelCustomOnline(2200113733),
		newLevelCustomOnline(2200116667),
		newLevelCustomOnline(2200118707),
		newLevelCustomOnline(2200119774),
		newLevelCustomOnline(2200122075),
		newLevelCustomOnline(2200123719),
		newLevelCustomOnline(2200126432),
		newLevelCustomOnline(2200129454),
		newLevelCustomOnline(2200131581),
		newLevelCustomOnline(2200137191),
		newLevelCustomOnline(2200140347),
		newLevelCustomOnline(2200142489),
		newLevelCustomOnline(2200145858),
		newLevelCustomOnline(2200147947),
		newLevelCustomOnline(2200152837),
		newLevelCustomOnline(2200157521),
		newLevelCustomOnline(2200161014),
		newLevelCustomOnline(2200163529),
		newLevelCustomOnline(2200166476),
		newLevelCustomOnline(2200561630),
		newLevelCustomOnline(2200566979),
		newLevelCustomOnline(2200572201),
		newLevelCustomOnline(2200577287),
		newLevelCustomOnline(2200582772),
		newLevelCustomOnline(2200585235),
		newLevelCustomOnline(2200614142),
		newLevelCustomOnline(2200631612),
		newLevelCustomOnline(2205919112),
		newLevelCustomOnline(2205950305),
		newLevelCustomOnline(2205969235),
		newLevelCustomOnline(2205978243),
		newLevelCustomOnline(2206006449),
		newLevelCustomOnline(2206027577),
		newLevelCustomOnline(2206041190),
		newLevelCustomOnline(2206047592),
		newLevelCustomOnline(2206225526),
		newLevelCustomOnline(2206344259),
		newLevelCustomOnline(2206360656),
		newLevelCustomOnline(2206388990),
		newLevelCustomOnline(2206397608),
		newLevelCustomOnline(2206407020),
		newLevelCustomOnline(2206417499),
		newLevelCustomOnline(2206431577),
		newLevelCustomOnline(2206435705),
		newLevelCustomOnline(2206453774),
		newLevelCustomOnline(2206460543),
		newLevelCustomOnline(2206464628),
		newLevelCustomOnline(2206467263),
		newLevelCustomOnline(2206518302),
		newLevelCustomOnline(2206539363),
		newLevelCustomOnline(2206542211),
		newLevelCustomOnline(2208826631),
		newLevelCustomOnline(2208831597),
		newLevelCustomOnline(2208836118),
		newLevelCustomOnline(2208843238),
		newLevelCustomOnline(2208847724),
		newLevelCustomOnline(2208851046),
		newLevelCustomOnline(2208859746),
		newLevelCustomOnline(2208864249),
		newLevelCustomOnline(2208897753),
		newLevelCustomOnline(2208910916),
		newLevelCustomOnline(2208914173),
		newLevelCustomOnline(2208916640),
		newLevelCustomOnline(2208919069),
		newLevelCustomOnline(2208928980),
		newLevelCustomOnline(2208932119),
		newLevelCustomOnline(2208933857),
		newLevelCustomOnline(2208943514),
		newLevelCustomOnline(2208946630),
		newLevelCustomOnline(2208996342),
		newLevelCustomOnline(2209006315),
		newLevelCustomOnline(2209010129),
		newLevelCustomOnline(2209020283),
		newLevelCustomOnline(2209033059),
		newLevelCustomOnline(2209046306),
		newLevelCustomOnline(2209407522),
		newLevelCustomOnline(2209422159),
		newLevelCustomOnline(2209838063),
		newLevelCustomOnline(2209860643),
		newLevelCustomOnline(2209878071),
		newLevelCustomOnline(2209902974),
		newLevelCustomOnline(2209906828),
	}

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
