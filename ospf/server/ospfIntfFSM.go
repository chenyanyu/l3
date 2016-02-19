package server

import (
    "fmt"
    "time"
    "l3/ospf/config"
    "encoding/binary"
)

func (server *OSPFServer) StartOspfIntfFSM(key IntfConfKey) {
        ent, _ := server.IntfConfMap[key]
        if ent.IfType == config.PointToPoint {
                server.StartOspfP2PIntfFSM(key)
        } else if ent.IfType == config.Broadcast {
                server.StartOspfBroadcastIntfFSM(key)
        }
}

func (server *OSPFServer) StartOspfP2PIntfFSM(key IntfConfKey) {
        server.StartSendHelloPkt(key)
        for {
                ent, _ := server.IntfConfMap[key]
                select {
                case <-ent.HelloIntervalTicker.C:
                        server.StartSendHelloPkt(key)
                case createMsg := <-ent.NeighCreateCh:
                    if bytesEqual(createMsg.DRtr, []byte{0, 0, 0, 0}) == false ||
                        bytesEqual(createMsg.BDRtr, []byte{0, 0, 0, 0}) == false {
                        server.logger.Err("DR or BDR is non zero")
                        continue
                    }
                        neighborKey := NeighborKey{
                                RouterId: createMsg.RouterId,
                        }
                        neighborEntry, exist := ent.NeighborMap[neighborKey]
                        if !exist {
                                neighborEntry.NbrIP = createMsg.NbrIP
                                neighborEntry.TwoWayStatus = createMsg.TwoWayStatus
                                neighborEntry.RtrPrio = createMsg.RtrPrio
                                neighborEntry.FullState = false
                                ent.NeighborMap[neighborKey] = neighborEntry
                                server.IntfConfMap[key] = ent
                                server.logger.Info(fmt.Sprintln("1 IntfConf neighbor entry", server.IntfConfMap[key].NeighborMap, "neighborKey:", neighborKey))
                        }
                case changeMsg:= <-ent.NeighChangeCh:
                    if bytesEqual(changeMsg.DRtr, []byte{0, 0, 0, 0}) == false ||
                        bytesEqual(changeMsg.BDRtr, []byte{0, 0, 0, 0}) == false {
                        server.logger.Err("DR or BDR is non zero")
                        continue
                    }
                    neighborKey := NeighborKey {
                        RouterId:       changeMsg.RouterId,
                    }
                    neighborEntry, exist := ent.NeighborMap[neighborKey]
                    if exist {
                        server.logger.Info(fmt.Sprintln("Change msg: ", changeMsg, "neighbor entry:", neighborEntry, "neighbor key:", neighborKey))
                        neighborEntry.NbrIP = changeMsg.NbrIP
                        neighborEntry.TwoWayStatus = changeMsg.TwoWayStatus
                        neighborEntry.RtrPrio = changeMsg.RtrPrio
                        neighborEntry.DRtr = changeMsg.DRtr
                        neighborEntry.BDRtr = changeMsg.BDRtr
                        ent.NeighborMap[neighborKey] = neighborEntry
                        server.IntfConfMap[key] = ent
                        server.logger.Info(fmt.Sprintln("2 IntfConf neighbor entry", server.IntfConfMap[key].NeighborMap))
                    } else {
                        server.logger.Err(fmt.Sprintln("Neighbor entry does not exists", neighborKey.RouterId))
                    }
                case nbrStateChangeMsg := <-ent.NbrStateChangeCh:
                    // Only when Neighbor Went Down from TwoWayStatus
                    server.logger.Info(fmt.Sprintf("Recev Neighbor State Change message", nbrStateChangeMsg))
                    server.processNbrDownEvent(nbrStateChangeMsg, key, true)
                case state := <-ent.FSMCtrlCh:
                        if state == false {
                                server.StopSendHelloPkt(key)
                                ent.FSMCtrlStatusCh <- false
                                return
                        }
                }
        }

}

func (server *OSPFServer) StartOspfBroadcastIntfFSM(key IntfConfKey) {
    server.StartSendHelloPkt(key)
    for {
        ent, _ := server.IntfConfMap[key]
        select {
        case <-ent.HelloIntervalTicker.C:
            server.StartSendHelloPkt(key)
        case <-ent.WaitTimer.C:
            server.logger.Info("Wait timer expired")
            //server.IntfConfMap[key] = ent
            // Elect BDR And DR
            server.ElectBDRAndDR(key)
        case msg := <-ent.BackupSeenCh:
            server.logger.Info(fmt.Sprintf("Transit to action state because of backup seen", msg))
            server.ElectBDRAndDR(key)
        case createMsg := <-ent.NeighCreateCh:
            neighborKey := NeighborKey {
                RouterId:       createMsg.RouterId,
                //NbrIP:          createMsg.NbrIP,
            }
            neighborEntry, exist := ent.NeighborMap[neighborKey]
            if !exist {
                neighborEntry.NbrIP = createMsg.NbrIP
                neighborEntry.TwoWayStatus = createMsg.TwoWayStatus
                neighborEntry.RtrPrio = createMsg.RtrPrio
                neighborEntry.DRtr = createMsg.DRtr
                neighborEntry.BDRtr = createMsg.BDRtr
                neighborEntry.FullState = false
                ent.NeighborMap[neighborKey] = neighborEntry
                server.IntfConfMap[key] = ent
                server.logger.Info(fmt.Sprintln("1 IntfConf neighbor entry", server.IntfConfMap[key].NeighborMap, "neighborKey:", neighborKey))
                if createMsg.TwoWayStatus == true &&
                    ent.IfFSMState > config.Waiting {
                    server.ElectBDRAndDR(key)
                }
            }
        case changeMsg:= <-ent.NeighChangeCh:
            neighborKey := NeighborKey {
                RouterId:       changeMsg.RouterId,
                //NbrIP:          changeMsg.NbrIP,
            }
            neighborEntry, exist := ent.NeighborMap[neighborKey]
            if exist {
                server.logger.Info(fmt.Sprintln("Change msg: ", changeMsg, "neighbor entry:", neighborEntry, "neighbor key:", neighborKey))
                //rtrId := changeMsg.RouterId
                NbrIP := changeMsg.NbrIP
                oldRtrPrio := neighborEntry.RtrPrio
                oldDRtr := binary.BigEndian.Uint32(neighborEntry.DRtr)
                oldBDRtr := binary.BigEndian.Uint32(neighborEntry.BDRtr)
                newDRtr := binary.BigEndian.Uint32(changeMsg.DRtr)
                newBDRtr := binary.BigEndian.Uint32(changeMsg.BDRtr)
                oldTwoWayStatus := neighborEntry.TwoWayStatus
                neighborEntry.NbrIP = changeMsg.NbrIP
                neighborEntry.TwoWayStatus = changeMsg.TwoWayStatus
                neighborEntry.RtrPrio = changeMsg.RtrPrio
                neighborEntry.DRtr = changeMsg.DRtr
                neighborEntry.BDRtr = changeMsg.BDRtr
                ent.NeighborMap[neighborKey] = neighborEntry
                server.IntfConfMap[key] = ent
                server.logger.Info(fmt.Sprintln("2 IntfConf neighbor entry", server.IntfConfMap[key].NeighborMap))
                if ent.IfFSMState > config.Waiting {
                    // RFC2328 Section 9.2 (Neighbor Change Event)
                    if (oldDRtr == NbrIP && newDRtr != NbrIP && oldTwoWayStatus == true) ||
                        (oldDRtr != NbrIP && newDRtr == NbrIP && oldTwoWayStatus == true) ||
                        (oldBDRtr == NbrIP && newBDRtr != NbrIP && oldTwoWayStatus == true) ||
                        (oldBDRtr != NbrIP && newBDRtr == NbrIP && oldTwoWayStatus == true) ||
                        (oldTwoWayStatus != changeMsg.TwoWayStatus) ||
                        (oldRtrPrio != changeMsg.RtrPrio && oldTwoWayStatus == true) {

                        // Update Neighbor and Re-elect BDR And DR
                        server.ElectBDRAndDR(key)
                    }
                }
            }
        case nbrStateChangeMsg := <-ent.NbrStateChangeCh:
            // Only when Neighbor Went Down from TwoWayStatus
            // Todo: Handle NbrIP: Ashutosh
            server.logger.Info(fmt.Sprintf("Recev Neighbor State Change message", nbrStateChangeMsg))
            server.processNbrDownEvent(nbrStateChangeMsg, key, false)
        case state := <-ent.FSMCtrlCh:
            if state == false {
                server.StopSendHelloPkt(key)
                ent.FSMCtrlStatusCh<-false
                return
            }
        case msg := <-ent.NbrFullStateCh:
            // Note : NBR State Machine should only send message if
            // NBR State changes to/from Full (but not to Down)
            server.processNbrFullStateMsg(msg, key)
        }
    }
}

func (server *OSPFServer)processNbrDownEvent(msg NbrStateChangeMsg,
    key IntfConfKey, p2p bool) {
    ent, _ := server.IntfConfMap[key]
    nbrKey := NeighborKey {
        RouterId:   msg.RouterId,
    }
    neighborEntry, exist := ent.NeighborMap[nbrKey]
    if exist {
        oldTwoWayStatus := neighborEntry.TwoWayStatus
        delete(ent.NeighborMap, nbrKey)
        server.logger.Info(fmt.Sprintln("Deleting", nbrKey))
        server.IntfConfMap[key] = ent
        if p2p == false {
                if ent.IfFSMState > config.Waiting {
                    // RFC2328 Section 9.2 (Neighbor Change Event)
                    if oldTwoWayStatus == true {
                        server.ElectBDRAndDR(key)
                    }
                }
        }
    }
}

// Nbr State machine has to send FullState change msg and then
// Send interface state down event
func (server *OSPFServer)processNbrFullStateMsg(msg NbrFullStateMsg,
    key IntfConfKey) {
    ent, _ := server.IntfConfMap[key]
    areaId := convertIPv4ToUint32(ent.IfAreaId)
    if msg.FullState == true {
        server.logger.Info("Neighbor State changed to full state")
    } else {
        server.logger.Info("Neighbor State changed from full state")
    }
    nbrKey := NeighborKey {
            RouterId:   msg.NbrRtrId,
    }
    nbrEntry, exist := ent.NeighborMap[nbrKey]
    if exist {
        if msg.FullState != nbrEntry.FullState &&
            ent.IfFSMState == config.DesignatedRouter {
            nbrEntry.FullState = msg.FullState
            ent.NeighborMap[nbrKey] = nbrEntry
            server.IntfConfMap[key] = ent
            lsaMsg := NetworkLSAChangeMsg {
                areaId: areaId,
                intfKey: key,
            }
            server.CreateNetworkLSACh <- lsaMsg
        }
    }
}

func (server *OSPFServer)ElectBDR(key IntfConfKey) ([]byte, uint32) {
    ent, _ := server.IntfConfMap[key]
    electedBDR := []byte {0, 0, 0, 0}
    var electedRtrPrio uint8
    var electedRtrId uint32
    var MaxRtrPrio uint8
    var RtrIdWithMaxPrio uint32
    var NbrIPWithMaxPrio uint32

    for nbrkey, nbrEntry := range ent.NeighborMap {
        if nbrEntry.TwoWayStatus == true &&
            nbrEntry.RtrPrio > 0 &&
            nbrEntry.NbrIP != 0 {
            tempDR := binary.BigEndian.Uint32(nbrEntry.DRtr)
            if tempDR == nbrEntry.NbrIP {
                continue
            }
            tempBDR := binary.BigEndian.Uint32(nbrEntry.BDRtr)
            if tempBDR == nbrEntry.NbrIP {
                if nbrEntry.RtrPrio > electedRtrPrio {
                    electedRtrPrio = nbrEntry.RtrPrio
                    electedRtrId = nbrkey.RouterId
                    electedBDR = nbrEntry.BDRtr
                } else if nbrEntry.RtrPrio == electedRtrPrio {
                    if electedRtrId < nbrkey.RouterId {
                        electedRtrPrio = nbrEntry.RtrPrio
                        electedRtrId = nbrkey.RouterId
                        electedBDR = nbrEntry.BDRtr
                    }
                }
            }
            if MaxRtrPrio < nbrEntry.RtrPrio {
                MaxRtrPrio = nbrEntry.RtrPrio
                RtrIdWithMaxPrio = nbrkey.RouterId
                NbrIPWithMaxPrio = nbrEntry.NbrIP
            } else if MaxRtrPrio == nbrEntry.RtrPrio {
                if RtrIdWithMaxPrio < nbrkey.RouterId {
                    MaxRtrPrio = nbrEntry.RtrPrio
                    RtrIdWithMaxPrio = nbrkey.RouterId
                    NbrIPWithMaxPrio = nbrEntry.NbrIP
                }
            }
        }
    }

    if ent.IfRtrPriority != 0 &&
        bytesEqual(ent.IfIpAddr.To4(), []byte {0, 0, 0, 0}) == false {
        if bytesEqual(ent.IfIpAddr.To4(), ent.IfDRIp) == false {
            if bytesEqual(ent.IfIpAddr.To4(), ent.IfBDRIp) == true {
                rtrId := binary.BigEndian.Uint32(server.ospfGlobalConf.RouterId)
                if ent.IfRtrPriority > electedRtrPrio {
                    electedRtrPrio = ent.IfRtrPriority
                    electedRtrId = rtrId
                    electedBDR = ent.IfIpAddr.To4()
                } else if ent.IfRtrPriority == electedRtrPrio {
                    if electedRtrId < rtrId {
                        electedRtrPrio = ent.IfRtrPriority
                        electedRtrId = rtrId
                        electedBDR = ent.IfIpAddr.To4()
                    }
                }
            }

            tempRtrId := binary.BigEndian.Uint32(server.ospfGlobalConf.RouterId)
            if MaxRtrPrio < ent.IfRtrPriority {
                MaxRtrPrio = ent.IfRtrPriority
                NbrIPWithMaxPrio = binary.BigEndian.Uint32(ent.IfIpAddr.To4())
                RtrIdWithMaxPrio = tempRtrId
            } else if MaxRtrPrio == ent.IfRtrPriority {
                if RtrIdWithMaxPrio < tempRtrId {
                    MaxRtrPrio = ent.IfRtrPriority
                    NbrIPWithMaxPrio = binary.BigEndian.Uint32(ent.IfIpAddr.To4())
                    RtrIdWithMaxPrio = tempRtrId
                }
            }

        }
    }
    if bytesEqual(electedBDR, []byte{0, 0, 0, 0}) == true {
        binary.BigEndian.PutUint32(electedBDR, NbrIPWithMaxPrio)
        electedRtrId = RtrIdWithMaxPrio
    }

    return electedBDR, electedRtrId
}

func (server *OSPFServer)ElectDR(key IntfConfKey, electedBDR []byte, electedBDRtrId  uint32) ([]byte, uint32) {
    ent, _ := server.IntfConfMap[key]
    electedDR := []byte {0, 0, 0, 0}
    var electedRtrPrio uint8
    var electedDRtrId uint32

    for key, nbrEntry := range ent.NeighborMap {
        if nbrEntry.TwoWayStatus == true &&
            nbrEntry.RtrPrio > 0  &&
            nbrEntry.NbrIP != 0 {
            tempDR := binary.BigEndian.Uint32(nbrEntry.DRtr)
            if tempDR == nbrEntry.NbrIP {
                if nbrEntry.RtrPrio > electedRtrPrio {
                    electedRtrPrio = nbrEntry.RtrPrio
                    electedDRtrId = key.RouterId
                    electedDR = nbrEntry.DRtr
                } else if nbrEntry.RtrPrio == electedRtrPrio {
                    if electedDRtrId < key.RouterId {
                        electedRtrPrio = nbrEntry.RtrPrio
                        electedDRtrId = key.RouterId
                        electedDR = nbrEntry.DRtr
                    }
                }
            }
        }
    }

    if ent.IfRtrPriority > 0 &&
        bytesEqual(ent.IfIpAddr.To4(), []byte {0, 0, 0, 0}) == false {
        if bytesEqual(ent.IfIpAddr.To4(), ent.IfDRIp) == true {
            rtrId := binary.BigEndian.Uint32(server.ospfGlobalConf.RouterId)
            if ent.IfRtrPriority > electedRtrPrio {
                electedRtrPrio = ent.IfRtrPriority
                electedDRtrId = rtrId
                electedDR = ent.IfIpAddr.To4()
            } else if ent.IfRtrPriority == electedRtrPrio {
                if electedDRtrId < rtrId {
                    electedRtrPrio = ent.IfRtrPriority
                    electedDRtrId = rtrId
                    electedDR = ent.IfIpAddr.To4()
                }
            }
        }
    }

    if bytesEqual(electedDR, []byte{0, 0, 0, 0}) == true {
        electedDR = electedBDR
        electedDRtrId = electedBDRtrId
    }
    return electedDR, electedDRtrId
}

func (server *OSPFServer)ElectBDRAndDR(key IntfConfKey) {
        ent, _ := server.IntfConfMap[key]
        server.logger.Info(fmt.Sprintln("Election of BDR andDR", ent.IfFSMState))

        oldDRtrId := ent.IfDRtrId
        oldBDRtrId := ent.IfBDRtrId
        //oldBDR := ent.IfBDRIp
        oldState := ent.IfFSMState
        var newState config.IfState

        electedBDR, electedBDRtrId := server.ElectBDR(key)
        ent.IfBDRIp = electedBDR
        ent.IfBDRtrId = electedBDRtrId
        electedDR, electedDRtrId := server.ElectDR(key, electedBDR, electedBDRtrId)
        ent.IfDRIp = electedDR
        ent.IfDRtrId = electedDRtrId
        if bytesEqual(ent.IfDRIp, ent.IfIpAddr.To4()) == true {
                newState = config.DesignatedRouter
        } else if bytesEqual(ent.IfBDRIp, ent.IfIpAddr.To4()) == true {
                newState = config.BackupDesignatedRouter
        } else {
                newState = config.OtherDesignatedRouter
        }

        server.logger.Info(fmt.Sprintln("1. Election of BDR:", ent.IfBDRIp, " and DR:", ent.IfDRIp, "new State:", newState, "DR Id:", ent.IfDRtrId, "BDR Id:", ent.IfBDRtrId))
        server.IntfConfMap[key] = ent

        if (newState != oldState &&
                !(newState == config.OtherDesignatedRouter &&
                    oldState < config.OtherDesignatedRouter)) {
                ent, _ = server.IntfConfMap[key]
                electedBDR, electedBDRtrId = server.ElectBDR(key)
                ent.IfBDRIp = electedBDR
                ent.IfBDRtrId = electedBDRtrId
                electedDR, electedDRtrId = server.ElectDR(key, electedBDR, electedBDRtrId)
                ent.IfDRIp = electedDR
                ent.IfDRtrId = electedDRtrId
                if bytesEqual(ent.IfDRIp, ent.IfIpAddr.To4()) == true {
                    newState = config.DesignatedRouter
                } else if bytesEqual(ent.IfBDRIp, ent.IfIpAddr.To4()) == true {
                    newState = config.BackupDesignatedRouter
                } else {
                    newState = config.OtherDesignatedRouter
                }
                server.logger.Info(fmt.Sprintln("2. Election of BDR:", ent.IfBDRIp, " and DR:", ent.IfDRIp, "new State:", newState, "DR Id:", ent.IfDRtrId, "BDR Id:", ent.IfBDRtrId))
                server.IntfConfMap[key] = ent
        }

        server.createAndSendEventsIntfFSM(key, oldState, newState, oldDRtrId, oldBDRtrId)
}

func (server *OSPFServer)createAndSendEventsIntfFSM(key IntfConfKey,
        oldState config.IfState, newState config.IfState, oldDRtrId uint32,
        oldBDRtrId uint32) {
        ent, _ := server.IntfConfMap[key]
        ent.IfFSMState = newState
        // Need to Check: do we need to add events even when we
        // come back to same state after DR or BDR Election
        ent.IfEvents = ent.IfEvents + 1
        server.IntfConfMap[key] = ent
        server.logger.Info(fmt.Sprintln("Final Election of BDR:", ent.IfBDRIp, " and DR:", ent.IfDRIp, "new State:", newState))

        areaId := convertIPv4ToUint32(ent.IfAreaId)
        msg := LSAChangeMsg {
                areaId: areaId,
        }

        msg1 := NetworkLSAChangeMsg {
                areaId: areaId,
                intfKey: key,
        }

        server.logger.Info("1. Sending msg for router LSA generation")
        server.IntfStateChangeCh <- msg

        if oldState != newState {
                if newState == config.DesignatedRouter {
                        // Construct Network LSA
                        server.logger.Info("1. Sending msg for Network LSA generation")
                        server.CreateNetworkLSACh <- msg1
                } else if oldState == config.DesignatedRouter {
                        // Flush Network LSA
                        server.logger.Info("2. Sending msg for Network LSA generation")
                        server.FlushNetworkLSACh <- msg1
                }
                server.logger.Info(fmt.Sprintln("oldState", oldState, " != newState", newState))
        }
        server.logger.Info(fmt.Sprintln("oldState", oldState, " newState", newState))

/*
        if oldDRtrId != ent.IfDRtrId {
                adjOKEvtMsg := AdjOKEvtMsg {
                        NewDRtrId:         ent.IfDRtrId,
                        OldDRtrId:         oldDRtrId,
                        NewBDRtrId:        ent.IfBDRtrId,
                        OldBDRtrId:        oldBDRtrId,
                }
                server.logger.Info("Intf State Machine: Sending AdjOK Event to NBR State Machine because of DR change")
                server.AdjOKEvtCh <- adjOKEvtMsg
        } else if oldBDRtrId != ent.IfBDRtrId {
                adjOKEvtMsg := AdjOKEvtMsg {
                        NewDRtrId:         ent.IfDRtrId,
                        OldDRtrId:         oldDRtrId,
                        NewBDRtrId:        ent.IfBDRtrId,
                        OldBDRtrId:        oldBDRtrId,
                }
                server.logger.Info("Intf State Machine: Sending AdjOK Event to NBR State Machine because of BDR change")
                server.AdjOKEvtCh <- adjOKEvtMsg
        }
*/
}


func (server *OSPFServer)StopOspfIntfFSM(key IntfConfKey) {
    ent, _ := server.IntfConfMap[key]
    ent.FSMCtrlCh<-false
    cnt := 0
    for {
        select {
        case status := <-ent.FSMCtrlStatusCh:
            if status == false { // False Means Trans Pkt Thread Stopped
                server.logger.Info("Stopped Sending Hello Pkt")
                return
            }
        default:
            time.Sleep(time.Duration(10) * time.Millisecond)
            cnt = cnt + 1
            if cnt == 100 {
                server.logger.Err("Unable to stop the Tx thread")
                return
            }
        }
    }
}

