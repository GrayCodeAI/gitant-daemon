// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";

/// @title GitantFeeDistributor
/// @notice Weekly fee distribution: 75% nodes, 24% stakers, 1% keeper
contract GitantFeeDistributor is Ownable {
    using SafeERC20 for IERC20;

    IERC20 public immutable token;

    uint256 public constant EPOCH_DURATION = 7 days;
    uint256 public constant NODE_SHARE_BPS = 7500;    // 75%
    uint256 public constant STAKER_SHARE_BPS = 2400;  // 24%
    uint256 public constant KEEPER_SHARE_BPS = 100;   // 1%

    address public nodeStakingContract;
    address public stakingContract;
    address public keeper;

    uint256 public lastDistribution;
    uint256 public totalDistributed;

    struct EpochInfo {
        uint256 totalFees;
        uint256 nodeShare;
        uint256 stakerShare;
        uint256 keeperShare;
        uint256 timestamp;
    }

    EpochInfo[] public epochs;

    event FeesDistributed(uint256 indexed epoch, uint256 totalFees);
    event EpochCompleted(uint256 indexed epoch, uint256 timestamp);

    constructor(
        address _token,
        address _nodeStakingContract,
        address _stakingContract,
        address _keeper
    ) Ownable(msg.sender) {
        token = IERC20(_token);
        nodeStakingContract = _nodeStakingContract;
        stakingContract = _stakingContract;
        keeper = _keeper;
    }

    /// @notice Distribute accumulated fees (anyone can call)
    function distribute() external {
        require(block.timestamp >= lastDistribution + EPOCH_DURATION, "Too early");

        uint256 balance = token.balanceOf(address(this));
        require(balance > 0, "No fees to distribute");

        uint256 nodeShare = (balance * NODE_SHARE_BPS) / 10000;
        uint256 stakerShare = (balance * STAKER_SHARE_BPS) / 10000;
        uint256 keeperShare = balance - nodeShare - stakerShare;

        // Send to node staking contract
        token.safeTransfer(nodeStakingContract, nodeShare);

        // Send to staking contract
        token.safeTransfer(stakingContract, stakerShare);

        // Send to keeper
        token.safeTransfer(keeper, keeperShare);

        uint256 epochId = epochs.length;
        epochs.push(EpochInfo({
            totalFees: balance,
            nodeShare: nodeShare,
            stakerShare: stakerShare,
            keeperShare: keeperShare,
            timestamp: block.timestamp
        }));

        lastDistribution = block.timestamp;
        totalDistributed += balance;

        emit FeesDistributed(epochId, balance);
        emit EpochCompleted(epochId, block.timestamp);
    }

    /// @notice Get epoch info
    function getEpoch(uint256 epochId) external view returns (EpochInfo memory) {
        return epochs[epochId];
    }

    /// @notice Get total epochs
    function getEpochCount() external view returns (uint256) {
        return epochs.length;
    }

    function setContracts(
        address _nodeStaking,
        address _staking,
        address _keeper
    ) external onlyOwner {
        nodeStakingContract = _nodeStaking;
        stakingContract = _staking;
        keeper = _keeper;
    }
}
