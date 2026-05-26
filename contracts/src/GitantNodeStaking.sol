// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";

/// @title GitantNodeStaking
/// @notice Proof-of-Stake for node operators with slashing
/// @dev Nodes must stake minimum tokens and maintain heartbeat
contract GitantNodeStaking is Ownable {
    using SafeERC20 for IERC20;

    struct NodeInfo {
        address operator;
        uint256 stake;
        uint256 lastHeartbeat;
        string multiaddr;
        bool active;
        uint256 slashCount;
        uint256 totalSlashed;
    }

    IERC20 public immutable token;

    uint256 public constant MIN_STAKE = 10_000 * 1e18;
    uint256 public constant HEARTBEAT_TIMEOUT = 24 hours;
    uint256 public constant SLASH_PERCENT_LIGHT = 10;   // 10%
    uint256 public constant SLASH_PERCENT_MEDIUM = 50;   // 50%
    uint256 public constant SLASH_PERCENT_HEAVY = 100;   // 100%

    mapping(string => NodeInfo) public nodes;
    string[] public activeNodes;
    mapping(address => string[]) public operatorNodes;
    mapping(string => bool) public isNodeActive;

    uint256 public totalNodeStake;
    address public slashingAuthority;

    event NodeRegistered(string indexed nodeId, address operator, uint256 stake);
    event NodeDeregistered(string indexed nodeId);
    event Heartbeat(string indexed nodeId, uint256 timestamp);
    event NodeSlashed(string indexed nodeId, uint256 amount, string reason);
    event NodeJailed(string indexed nodeId, uint256 until);

    constructor(address _token, address _slashingAuthority) Ownable(msg.sender) {
        token = IERC20(_token);
        slashingAuthority = _slashingAuthority;
    }

    /// @notice Register a new node
    /// @param nodeId Unique node identifier
    /// @param multiaddr libp2p multiaddr
    /// @param stakeAmount Amount to stake
    function registerNode(
        string calldata nodeId,
        string calldata multiaddr,
        uint256 stakeAmount
    ) external {
        require(stakeAmount >= MIN_STAKE, "Insufficient stake");
        require(!isNodeActive[nodeId], "Node already registered");

        token.safeTransferFrom(msg.sender, address(this), stakeAmount);

        nodes[nodeId] = NodeInfo({
            operator: msg.sender,
            stake: stakeAmount,
            lastHeartbeat: block.timestamp,
            multiaddr: multiaddr,
            active: true,
            slashCount: 0,
            totalSlashed: 0
        });

        activeNodes.push(nodeId);
        operatorNodes[msg.sender].push(nodeId);
        isNodeActive[nodeId] = true;
        totalNodeStake += stakeAmount;

        emit NodeRegistered(nodeId, msg.sender, stakeAmount);
    }

    /// @notice Deregister a node and reclaim stake
    /// @param nodeId The node ID
    function deregisterNode(string calldata nodeId) external {
        NodeInfo storage node = nodes[nodeId];
        require(node.active, "Node not active");
        require(node.operator == msg.sender, "Not operator");

        node.active = false;
        isNodeActive[nodeId] = false;
        totalNodeStake -= node.stake;

        token.safeTransfer(msg.sender, node.stake - node.totalSlashed);

        emit NodeDeregistered(nodeId);
    }

    /// @notice Send heartbeat to prove node is alive
    /// @param nodeId The node ID
    function heartbeat(string calldata nodeId) external {
        NodeInfo storage node = nodes[nodeId];
        require(node.active, "Node not active");
        require(node.operator == msg.sender, "Not operator");

        node.lastHeartbeat = block.timestamp;

        emit Heartbeat(nodeId, block.timestamp);
    }

    /// @notice Slash a node for misbehavior
    /// @param nodeId The node ID
    /// @param severity 0=light, 1=medium, 2=heavy
    /// @param reason Reason for slashing
    function slash(string calldata nodeId, uint8 severity, string calldata reason) external {
        require(msg.sender == slashingAuthority || msg.sender == owner(), "Not authorized");

        NodeInfo storage node = nodes[nodeId];
        require(node.active, "Node not active");

        uint256 slashPercent;
        if (severity == 0) slashPercent = SLASH_PERCENT_LIGHT;
        else if (severity == 1) slashPercent = SLASH_PERCENT_MEDIUM;
        else slashPercent = SLASH_PERCENT_HEAVY;

        uint256 slashAmount = (node.stake * slashPercent) / 100;
        node.stake -= slashAmount;
        node.totalSlashed += slashAmount;
        node.slashCount++;
        totalNodeStake -= slashAmount;

        if (node.stake < MIN_STAKE) {
            node.active = false;
            isNodeActive[nodeId] = false;
        }

        emit NodeSlashed(nodeId, slashAmount, reason);
    }

    /// @notice Check if a node has missed heartbeat
    /// @param nodeId The node ID
    /// @return true if node is stale
    function isStale(string calldata nodeId) external view returns (bool) {
        NodeInfo storage node = nodes[nodeId];
        return node.active && block.timestamp > node.lastHeartbeat + HEARTBEAT_TIMEOUT;
    }

    /// @notice Get all active nodes
    function getActiveNodes() external view returns (string[] memory) {
        return activeNodes;
    }

    /// @notice Get nodes for an operator
    function getOperatorNodes(address operator) external view returns (string[] memory) {
        return operatorNodes[operator];
    }

    function setSlashingAuthority(address _authority) external onlyOwner {
        slashingAuthority = _authority;
    }
}
