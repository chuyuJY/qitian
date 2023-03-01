package geecache

import "qitian/geeCache/geeCachePb/pb"

type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool) // 根据传入的 key 选择相应节点 PeerGetter
}

type PeerGetter interface {
	Get(in *pb.Request, out *pb.Response) error // 从对应 group 查找缓存值
}
