package manager

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// spigotURLRegex 匹配 SpigotMC 资源页面 URL 并提取数字 Resource ID
// 支持：
//   - https://www.spigotmc.org/resources/essentialsx.9089/
//   - https://spigotmc.org/resources/vault.34315
//   - https://spigotmc.org/resources/34315/
//   - https://www.spigotmc.org/resources/some-name.12345/updates
var spigotURLRegex = regexp.MustCompile(`(?i)spigotmc\.org/resources/(?:[^/?#]+\.)?(\d+)`)

// ExtractSpigotResourceID 从插件的 website 网址中尝试提取 SpigotMC 资源数字 ID
// 若网址为空、非 SpigotMC 地址或无法提取有效数字，则返回空字符串。
func ExtractSpigotResourceID(website string) string {
	if website == "" {
		return ""
	}
	matches := spigotURLRegex.FindStringSubmatch(website)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// updateHTTPClient 设置 3 秒超时，防止网络缓慢阻塞后台协程
var updateHTTPClient = &http.Client{
	Timeout: 3 * time.Second,
}

// FetchSpigotLatestVersion 请求 SpigotMC 官方 Update API 获取最新版本号纯文本
func FetchSpigotLatestVersion(resourceID string) (string, error) {
	if resourceID == "" {
		return "", fmt.Errorf("resource id is empty")
	}

	apiURL := "https://api.spigotmc.org/legacy/update.php?resource=" + resourceID
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}

	// 设置辨识度高的 User-Agent，避免被防护规则拦截
	req.Header.Set("User-Agent", "PlugLever/1.0 (Minecraft Server Plugin Manager)")

	resp, err := updateHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("spigot api returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return "", err
	}

	version := strings.TrimSpace(string(body))
	if version == "" || len(version) > 50 || strings.ContainsAny(version, "<>\r\n") {
		return "", fmt.Errorf("invalid version received: %s", version)
	}

	return version, nil
}

// CheckSpigotUpdate 综合检测某个插件是否有 SpigotMC 新版本。
// 返回:
//   - latestVer: 远端最新版本号
//   - hasUpdate: 是否存在比当前本地版本更新的版本
// 任意步骤出错或未发现新版时，静默返回 ("", false)。
func CheckSpigotUpdate(website, currentVersion string) (string, bool) {
	resourceID := ExtractSpigotResourceID(website)
	if resourceID == "" {
		return "", false
	}

	remoteVer, err := FetchSpigotLatestVersion(resourceID)
	if err != nil || remoteVer == "" {
		return "", false
	}

	if IsNewerVersion(currentVersion, remoteVer) {
		return remoteVer, true
	}

	return "", false
}

// IsNewerVersion 比对当前版本与远端版本，判断远端是否为更新版本
func IsNewerVersion(current, remote string) bool {
	cNorm := cleanVersionString(current)
	rNorm := cleanVersionString(remote)

	if cNorm == "" || rNorm == "" || strings.EqualFold(cNorm, rNorm) {
		return false
	}

	cParts, cPre := splitVersionAndPreRelease(cNorm)
	rParts, rPre := splitVersionAndPreRelease(rNorm)

	maxLen := len(cParts)
	if len(rParts) > maxLen {
		maxLen = len(rParts)
	}

	// 逐段数字比对
	for i := 0; i < maxLen; i++ {
		cV := 0
		if i < len(cParts) {
			cV = cParts[i]
		}
		rV := 0
		if i < len(rParts) {
			rV = rParts[i]
		}

		if rV > cV {
			return true
		}
		if rV < cV {
			return false
		}
	}

	// 数字完全相同的前提下，检查预发布/快照标识
	// 如果当前版本是快照/预览版（带有 pre-release），而远端是正式版，则远端更新
	if cPre != "" && rPre == "" {
		return true
	}

	return false
}

// cleanVersionString 清洗版本号前缀
func cleanVersionString(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	return strings.TrimSpace(v)
}

// splitVersionAndPreRelease 将版本号拆分为数字段和预发布标识
// 例如: "1.20.4-SNAPSHOT" -> [1, 20, 4], "SNAPSHOT"
func splitVersionAndPreRelease(v string) ([]int, string) {
	pre := ""
	dashIdx := strings.Index(v, "-")
	mainPart := v
	if dashIdx != -1 {
		mainPart = v[:dashIdx]
		pre = v[dashIdx+1:]
	}

	subparts := strings.Split(mainPart, ".")
	var nums []int
	for _, s := range subparts {
		s = strings.TrimSpace(s)
		n, err := strconv.Atoi(s)
		if err == nil {
			nums = append(nums, n)
		} else {
			// 处理夹带非数字的情况（如 1.20b）
			var digits []rune
			for _, r := range s {
				if r >= '0' && r <= '9' {
					digits = append(digits, r)
				} else {
					break
				}
			}
			if len(digits) > 0 {
				if val, err := strconv.Atoi(string(digits)); err == nil {
					nums = append(nums, val)
				}
			}
		}
	}
	return nums, pre
}
