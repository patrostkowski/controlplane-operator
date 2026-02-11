package pki

const maxCommonNameLength = 64

func commonNameFromPod(podName string) (string, bool) {
	if len(podName) > maxCommonNameLength {
		return podName[:64], true
	}
	return podName, false
}
