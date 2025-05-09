package lib

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	configparser "github.com/alyu/configparser"
)

// sections in config file
const (
	CREDSection string = "Credentials"

	BucketEndpointSection string = "Bucket-Endpoint"

	BucketCnameSection string = "Bucket-Cname"

	AkServiceSection string = "AkService"

	DefaultSection string = "Default"
)

// config items in section AKSerivce
const (
	ItemEcsAk string = "ecsAk"
)

// config items in section Credentials
const (
	ItemRamRoleArn string = "ramRoleArn"
	ItemExternalId string = "externalId"
)

type configOption struct {
	showNames     []string
	cfInteractive bool
	reveal        bool
	helpChinese   string
	helpEnglish   string
}

// CredOptionList is all options in Credentials section
var CredOptionList = []string{
	OptionLanguage,
	OptionEndpoint,
	OptionAccessKeyID,
	OptionAccessKeySecret,
	OptionSTSToken,
	OptionOutputDir,
	OptionRamRoleArn,
	OptionUserAgent,
}

// CredOptionMap allows alias name for options in Credentials section
// name, allow to show in screen
var CredOptionMap = map[string]configOption{
	OptionLanguage:        configOption{[]string{"language", "Language"}, false, true, "", ""},
	OptionEndpoint:        configOption{[]string{"endpoint", "host"}, true, true, "", ""},
	OptionAccessKeyID:     configOption{[]string{"accessKeyID", "accessKeyId", "AccessKeyID", "AccessKeyId", "access_key_id", "access_id", "accessid", "access-key-id", "access-id"}, true, false, "", ""},
	OptionAccessKeySecret: configOption{[]string{"accessKeySecret", "AccessKeySecret", "access_key_secret", "access_key", "accesskey", "access-key-secret", "access-key"}, true, false, "", ""},
	OptionSTSToken:        configOption{[]string{"stsToken", "ststoken", "STSToken", "sts_token", "sts-token"}, true, false, "", ""},
	OptionOutputDir:       configOption{[]string{"outputDir", "output-dir", "output_dir", "output_directory"}, false, true, "ossutil生成的文件的输出目录, ", "the directory to store files generated by ossutil, "},
	OptionMode:            configOption{[]string{"mode", "Mode"}, false, false, "", ""},
	OptionRamRoleArn:      configOption{[]string{"ramRoleArn", "RamRoleArn", "ramrolearn", "ram_role_arn", "ram-role-arn"}, false, false, "", ""},
	OptionRoleSessionName: configOption{[]string{"roleSessionName", "RoleSessionName", "rolesessionname", "role-session-name", "role_session_name"}, false, false, "", ""},
	OptionExternalId:      configOption{[]string{"externalID", "externalId", "externalid", "external-id", "external_id"}, false, false, "", ""},
	OptionTokenTimeout:    configOption{[]string{"tokenTimeOut", "tokenTimeout", "tokentimeout", "token_timeout", "token-timeout"}, false, false, "", ""},
	OptionSTSRegion:       configOption{[]string{"stsRegion", "stsregion", "sts-region", "sts_region"}, false, false, "", ""},
	OptionECSRoleName:     configOption{[]string{"ecsRoleName", "EcsRoleName", "ecsrolename", "ecs-role-name", "ecs_role_name"}, false, false, "", ""},
}

// DefaultOptionMap allows alias name for options in default section
// name, allow to show in screen
var DefaultOptionMap = map[string]configOption{
	OptionUserAgent:      configOption{[]string{"userAgent", "useragent", "user-agent", "user_agent"}, false, false, "", ""},
	OptionLogLevel:       configOption{[]string{"loglevel", "log-level", "log_level"}, false, false, "", ""},
	OptionProxyHost:      configOption{[]string{"proxyHost", "proxyhost", "proxy-host", "proxy_host"}, false, false, "", ""},
	OptionProxyUser:      configOption{[]string{"proxyUser", "proxyuser", "proxy-user", "proxy_user"}, false, false, "", ""},
	OptionProxyPwd:       configOption{[]string{"proxyPwd", "proxypwd", "proxy-pwd", "proxy_pwd"}, false, false, "", ""},
	OptionReadTimeout:    configOption{[]string{"readTimeOut", "readtimeout", "read-timeout", "read_timeout"}, false, false, "", ""},
	OptionConnectTimeout: configOption{[]string{"connectTimeOut", "connectTimeout", "connecttimeout", "connect-timeout", "connect_timeout"}, false, false, "", ""},
	OptionRetryTimes:     configOption{[]string{"retryTimes", "retrytimes", "retry-times", "retry_times"}, false, false, "", ""},
}

// DecideConfigFile return the config file, if user not specified, return default one
func DecideConfigFile(configFile string) string {
	if configFile == "" {
		configFile = DefaultConfigFile
	}

	if len(configFile) >= 2 && strings.HasPrefix(configFile, "~"+string(os.PathSeparator)) {
		homeDir := currentHomeDir()
		if homeDir != "" {
			configFile = strings.Replace(configFile, "~", homeDir, 1)
		}
	}
	return configFile
}

// LoadConfig load the specified config file
func LoadConfig(configFile string) (OptionMapType, error) {
	var configMap OptionMapType
	var err error
	configMap, err = readConfigFromFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("Read config file error: %s, please try \"help config\" to set configuration or use \"--config-file\" option", err)
	}
	if err = checkConfig(configMap); err != nil {
		return nil, err
	}
	return configMap, nil
}

func readConfigFromFile(configFile string) (OptionMapType, error) {
	configFile = DecideConfigFile(configFile)

	config, err := configparser.Read(configFile)
	if err != nil {
		return nil, err
	}

	configMap := OptionMapType{}

	// get options in Default Section
	defaultSection, err := config.Section(DefaultSection)
	if err == nil {
		defaultOptions := defaultSection.Options()
		for name, option := range defaultOptions {
			if opName, ok := getOptionNameByDefault(strings.TrimSpace(name)); ok {
				configMap[strings.TrimSpace(opName)] = strings.TrimSpace(option)
			}
		}
	}

	// get options in cred section
	credSection, err := config.Section(CREDSection)
	if err != nil {
		return nil, err
	}

	credOptions := credSection.Options()

	//added
	//configMap[CREDSection] = map[string]string{}

	for name, option := range credOptions {
		if opName, ok := getOptionNameByStr(strings.TrimSpace(name)); ok {
			configMap[strings.TrimSpace(opName)] = strings.TrimSpace(option)
		} else {
			configMap[strings.TrimSpace(name)] = strings.TrimSpace(option)
		}
	}

	// get options in pair sections
	for _, sec := range []string{BucketEndpointSection, BucketCnameSection} {
		if section, err := config.Section(sec); err == nil {
			configMap[sec] = map[string]string{}
			options := section.Options()
			for bucket, host := range options {
				(configMap[sec]).(map[string]string)[strings.TrimSpace(bucket)] = strings.TrimSpace(host)
			}
		}
	}

	// get options in AKService for user-defined GetAk
	sec := AkServiceSection
	if section, err := config.Section(sec); err == nil {
		configMap[sec] = map[string]string{}
		options := section.Options()
		for ecsUrl, strUrl := range options {
			(configMap[sec]).(map[string]string)[strings.TrimSpace(ecsUrl)] = strings.TrimSpace(strUrl)
		}
	}

	return configMap, nil
}

// get loglevel from config file
func readLoglevelFromFile(configFile string) (string, error) {
	configFile = DecideConfigFile(configFile)
	config, err := configparser.Read(configFile)
	if err != nil {
		return "", err
	}
	sectionNameList := []string{CREDSection, DefaultSection}
	logConfig := DefaultOptionMap[OptionLogLevel]
	for _, sectionName := range sectionNameList {
		section, err := config.Section(sectionName)
		if err != nil {
			continue
		}
		for _, name := range logConfig.showNames {
			val := section.ValueOf(name)
			if val != "" {
				return val, nil
			}
		}
	}
	return "", nil
}

func getOptionNameByStr(name string) (string, bool) {
	for optionName, option := range CredOptionMap {
		for _, val := range option.showNames {
			if strings.EqualFold(name, val) {
				return optionName, true
			}
		}
	}
	return "", false
}

func getOptionNameByDefault(name string) (string, bool) {
	for optionName, option := range DefaultOptionMap {
		for _, val := range option.showNames {
			if strings.EqualFold(name, val) {
				return optionName, true
			}
		}
	}
	return "", false
}

func checkConfig(configMap OptionMapType) error {
	for name, opval := range configMap {
		if option, ok := OptionMap[name]; ok {
			if option.optionType == OptionTypeInt64 {
				if _, err := strconv.ParseInt(opval.(string), 10, 64); err != nil {
					return fmt.Errorf("error value of option \"%s\", the value is: %s in config file, which needs int64 type", name, opval)
				}
			}
			if option.optionType == OptionTypeAlternative {
				vals := strings.Split(option.minVal, "/")
				if FindPosCaseInsen(opval.(string), vals) == -1 {
					return fmt.Errorf("error value of option \"%s\", the value is: %s in config file, which is not anyone of %s", name, opval, option.minVal)
				}
			}
		}
	}
	return nil
}
