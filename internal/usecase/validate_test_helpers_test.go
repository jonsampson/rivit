package usecase

import "context"

type probeResult struct {
	exists bool
	err    error
}

type memoryProbe struct {
	paths     map[string]probeResult
	remotes   map[string]string
	remoteErr map[string]error
}

func (p memoryProbe) PathExists(_ context.Context, path string) (bool, error) {
	if res, ok := p.paths[path]; ok {
		return res.exists, res.err
	}
	return false, nil
}

func (p memoryProbe) OriginRemote(_ context.Context, repoPath string) (string, error) {
	if err, ok := p.remoteErr[repoPath]; ok {
		return "", err
	}
	return p.remotes[repoPath], nil
}
