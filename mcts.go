package gomcts

import (
	"crypto/rand"
	"math"
	"math/big"
	"sync"
)

type monteCarloTreeSearchGameNode struct {
	parent         *monteCarloTreeSearchGameNode
	children       []*monteCarloTreeSearchGameNode
	value          GameState
	untriedActions []Action
	causingAction  Action
	q              float64
	n              float64
	lock           sync.Mutex
}

// MonteCarloTreeSearch - function starting Monte Carlo Tree Search over provided GameState using RolloutPolicy of your choice, repeating simulation requested amount of time
func MonteCarloTreeSearch(state GameState, rolloutPolicy RolloutPolicy, simulations int) Action {
	root := newMCTSNode(nil, state, nil)
	var leaf *monteCarloTreeSearchGameNode
	for i := 0; i < simulations; i++ {
		leaf = root.treePolicy()
		go func() {
			result := leaf.rollout(rolloutPolicy)
			root.lock.Lock()
			leaf.backpropagate(result)
			root.lock.Unlock()
		}()
	}
	// return root.mostVisited().causingAction
	return root.uctBestChild(0.0).causingAction
}

func newMCTSNode(parentNode *monteCarloTreeSearchGameNode, state GameState, causingAction Action) monteCarloTreeSearchGameNode {
	node := monteCarloTreeSearchGameNode{parent: parentNode, value: state, causingAction: causingAction}
	node.children = make([]*monteCarloTreeSearchGameNode, 0, 0)
	node.untriedActions = state.GetLegalActions()
	return node
}

func rootMCTSNode(state GameState) monteCarloTreeSearchGameNode {
	return newMCTSNode(nil, state, nil)
}

func (node *monteCarloTreeSearchGameNode) mostVisited() (chosen *monteCarloTreeSearchGameNode) {
	maxValue := -math.MaxFloat64
	chosen = nil
	for _, child := range node.children {
		if child.n > maxValue {
			chosen = child
		}
	}
	return
}

func (node *monteCarloTreeSearchGameNode) uctBestChild(c float64) *monteCarloTreeSearchGameNode {
	// chosenIndex := 0
	maxValue := -math.MaxFloat64
	bestChildren := []*monteCarloTreeSearchGameNode{}
	for _, child := range node.children {
		v := 100000000.0
		if child.n > 0 {
			v = (child.q / child.n) + c*math.Sqrt(2*math.Log(node.n)/child.n)
		}
		if v > maxValue {
			maxValue = v
			// chosenIndex = i
			bestChildren = nil
			bestChildren = append(bestChildren, child)
		} else if v == maxValue {
			bestChildren = append(bestChildren, child)
		}
		//if c == 0.0 {
		//	log.Printf("Visits: %.0f Q-Value: %.3f\n", child.n, child.q)
		//}
	}
	n := len(bestChildren)
	index, _ := rand.Int(rand.Reader, big.NewInt(int64(n)))
	return bestChildren[index.Int64()]
	// return node.children[chosenIndex]
}

func (node *monteCarloTreeSearchGameNode) rollout(policy RolloutPolicy) GameResult {
	currentState := node.value.Clone()
	currentState.RandomizeUnknowns()
	for !currentState.IsGameEnded() {
		currentState = policy(currentState).ApplyTo(currentState)
	}
	gameResult, _ := currentState.EvaluateGame()
	return gameResult
}

func (node *monteCarloTreeSearchGameNode) backpropagate(result GameResult) {
	for !node.isRoot() {
		node.q += float64(result) * float64(node.parent.value.NextToMove())
		node.n++
		node = node.parent
	}
	node.n++
}

func (node *monteCarloTreeSearchGameNode) isTerminal() bool {
	_, ended := node.value.EvaluateGame()
	return ended
}

func (node *monteCarloTreeSearchGameNode) isFullyExpanded() bool {
	return len(node.untriedActions) == 0
}

func (node *monteCarloTreeSearchGameNode) popFirstUntriedAction() Action {
	action := node.untriedActions[0]
	node.untriedActions = node.untriedActions[1:]
	return action
}

func (node *monteCarloTreeSearchGameNode) expand() *monteCarloTreeSearchGameNode {
	action := node.popFirstUntriedAction()
	expandedChild := newMCTSNode(node, action.ApplyTo(node.value), action)
	node.addChild(&expandedChild)
	return &expandedChild
}

func (node *monteCarloTreeSearchGameNode) treePolicy() *monteCarloTreeSearchGameNode {
	for !node.isTerminal() {
		if !node.isFullyExpanded() {
			return node.expand()
		}
		node = node.uctBestChild(1.4)
	}
	return node
}
