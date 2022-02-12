package defs

const JRASP_DAEMON_VERSION = "1.0.1"

const DATE_FORMAT = "2006-01-02 15:04:05.000"

// jvm-rasp logo
const LOGO = "      ____      ____  ___       _____            _____ _____\n      | \\ \\    / |  \\/  |      |  __ \\    /\\    / ____|  __ \\\n      | |\\ \\  / /| \\  / |______| |__) |  /  \\  | (___ | |__) |\n  _   | | \\ \\/ / | |\\/| |______|  _  /  / /\\ \\  \\___ \\|  ___/\n | |__| |  \\  /  | |  | |      | | \\ \\ / ____ \\ ____) | |\n  \\____/    \\/   |_|  |_|      |_|  \\_/_/    \\_|_____/|_|\n"

const START_LOG_ID = 1000

const (
	START_UP              int = START_LOG_ID + 0
	LOG_VALUE             int = START_LOG_ID + 1
	ENV_VALUE             int = START_LOG_ID + 2
	CONFIG_VALUE          int = START_LOG_ID + 3
	DEBUG_PPROF           int = START_LOG_ID + 4
	HTTP_TOKEN            int = START_LOG_ID + 5
	ATTACH_DEFAULT        int = START_LOG_ID + 6
	ATTACH_READ_TOKEN     int = START_LOG_ID + 7
	LOAD_JAR              int = START_LOG_ID + 8
	UTILS_OpenFiles       int = START_LOG_ID + 9
	WATCH_DEFAULT         int = START_LOG_ID + 10
	HEART_BEAT            int = START_LOG_ID + 11  // 心跳
	NACOS_INIT            int = START_LOG_ID + 12
	NACOS_LISTEN_CONFIG   int = START_LOG_ID + 13
	OSS_DOWNLOAD          int = START_LOG_ID + 14
	OSS_UPLOAD            int = START_LOG_ID + 15
	DEPENDENCY_INFO       int = START_LOG_ID + 16
	AGENT_SUCCESS_EXIT    int = START_LOG_ID + 17  // agent 卸载成功
	JAVA_PROCESS_STARTUP  int = START_LOG_ID + 18  // 发现java启动
	JAVA_PROCESS_SHUTDOWN int = START_LOG_ID + 19  // 发现java退出
	AGENT_SUCCESS_INIT    int = START_LOG_ID + 20  // agent 加载成功(attach成功)
)
