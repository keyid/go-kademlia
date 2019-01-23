package main

import (
    "fmt"
    log "github.com/sirupsen/logrus"
    "kademlia/Dispatcher"
    ds "kademlia/datastructure"
    "kademlia/message"
    "net"
)

func (d *DHT) OnGetPeersRequest(msg *message.GetPeersRequest, addr net.UDPAddr) {
    log.Debug("OnGetPeersRequest")
    var data []byte

    peers := d.peerStore.Get(msg.InfoHash)
    if len(peers) > 0 {
        data = message.GetPeersResponse{
            T:     msg.T,
            Id:    d.selfNodeID,
            Token: message.Token("sdhh"),
            Peers: peers,
        }.Encode()
    } else {
        data = message.GetPeersResponseWithNodes{
            T:     msg.T,
            Id:    d.selfNodeID,
            Token: message.Token("sdhh"),
            Nodes: d.routingTable.GetK(msg.Id),
        }.Encode()
    }

    if _, err := d.conn.WriteToUDP(data, &addr); err != nil {
        log.Error("Failed to send GetPeersWithNodes response")
    }
}

func (d *DHT) onGetPeersResponse(infoHash ds.InfoHash, getPeers *message.GetPeersResponse, addr net.UDPAddr) {
    fmt.Println("!!! onGetPeersResponse !!!")

    d.peerStore.Add(infoHash, getPeers.Peers)
}

func (d *DHT) onGetPeersWithNodesResponse(infoHash ds.InfoHash, getPeersWithNodes *message.GetPeersResponseWithNodes, addr net.UDPAddr) {
    log.Debug("getPeersWithNodes", addr)

    for _, c := range getPeersWithNodes.Nodes {
        d.Insert(c)
    }

    if !d.peerStore.Contains(infoHash) {
        d.getPeersByNodes(infoHash, getPeersWithNodes.Nodes)
    }
}

func (d *DHT) OnGetPeers(infoHash ds.InfoHash, msg message.Message, addr net.UDPAddr) {
    switch v := msg.(type) {
    case *message.GetPeersResponseWithNodes:
        d.onGetPeersWithNodesResponse(infoHash, v, addr)
    case *message.GetPeersResponse:
        d.onGetPeersResponse(infoHash, v, addr)
    default:
        log.Debug("Error : default case for OnGetPeers")
        return
    }
}

func (d *DHT) getPeersByNodes(infoHash ds.InfoHash, nodes []ds.Node) {
    tx := message.NewTransactionId()

    for _, node := range nodes {
        getPeersRequest := message.GetPeersRequest{
            T:        tx,
            Id:       d.selfNodeID,
            InfoHash: infoHash,
        }
        node.Send(d.conn, getPeersRequest.Encode())
    }

    d.eventDispatcher.AddEvent(tx.String(), Dispatcher.Event{
        Duplicates: len(nodes),
        OnResponse: Dispatcher.NewCallback(d.OnGetPeers, infoHash),
    })
}

func (d *DHT) GetPeers(infoHash ds.InfoHash) {
    nodes := d.routingTable.GetK(infoHash)
    d.getPeersByNodes(infoHash, nodes)
}