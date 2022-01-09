package common

import "os"

var Sig = make(chan os.Signal, 1)

const JRASP_DAEMON_VERSION = "1.0.1"

const DATE_FORMAT = "2006-01-02 15:04:05.000"

// jvm-rasp logo
const LOGO = "      ____      ____  ___       _____            _____ _____\n      | \\ \\    / |  \\/  |      |  __ \\    /\\    / ____|  __ \\\n      | |\\ \\  / /| \\  / |______| |__) |  /  \\  | (___ | |__) |\n  _   | | \\ \\/ / | |\\/| |______|  _  /  / /\\ \\  \\___ \\|  ___/\n | |__| |  \\  /  | |  | |      | | \\ \\ / ____ \\ ____) | |\n  \\____/    \\/   |_|  |_|      |_|  \\_/_/    \\_|_____/|_|\n"

const (
	START_UP            int = 0
	LOG_VALUE           int = 1
	ENV_VALUE           int = 2
	CONFIG_VALUE        int = 3
	DEBUG_PPROF         int = 4
	HTTP_TOKEN          int = 5
	ATTACH_DEFAULT      int = 6
	ATTACH_READ_TOKEN   int = 7
	LOAD_JAR            int = 8
	UTILS_OpenFiles     int = 9
	WATCH_DEFAULT       int = 10
	HEART_BEAT          int = 11
	NACOS_INIT          int = 12
	NACOS_LISTEN_CONFIG int = 13
	OSS_DOWNLOAD        int = 14
	OSS_UPLOAD          int = 15
)
