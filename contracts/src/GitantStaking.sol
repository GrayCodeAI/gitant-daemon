// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";

/// @title GitantStaking
/// @notice Tier-weighted passive staking for $GITLAWB token
/// @dev Stakers earn protocol fees based on tier and stake duration
contract GitantStaking is Ownable {
    using SafeERC20 for IERC20;

    enum Tier {
        Observer,   // 0 stake - read-only
        Light,      // 1,000+ - serve reads, DHT
        Full,       // 10,000+ - accept pushes, issue certs
        Validator   // 100,000+ - governance, slashing
    }

    struct StakeInfo {
        uint256 amount;
        uint256 stakedAt;
        Tier tier;
        uint256 rewardDebt;
        bool active;
    }

    IERC20 public immutable token;

    uint256 public constant MIN_LIGHT = 1_000 * 1e18;
    uint256 public constant MIN_FULL = 10_000 * 1e18;
    uint256 public constant MIN_VALIDATOR = 100_000 * 1e18;

    uint256 public constant TIER_MULTIPLIER_LIGHT = 1;
    uint256 public constant TIER_MULTIPLIER_FULL = 4;
    uint256 public constant TIER_MULTIPLIER_VALIDATOR = 8;

    mapping(address => StakeInfo) public stakes;
    address[] public stakers;
    mapping(address => bool) public isStaker;

    uint256 public totalStaked;
    uint256 public rewardPool;
    uint256 public lastRewardTimestamp;
    uint256 public rewardRatePerSecond; // wei per second per staked token

    event Staked(address indexed user, uint256 amount, Tier tier);
    event Unstaked(address indexed user, uint256 amount);
    event RewardClaimed(address indexed user, uint256 amount);
    event TierUpgraded(address indexed user, Tier newTier);

    constructor(address _token) Ownable(msg.sender) {
        token = IERC20(_token);
    }

    /// @notice Stake tokens
    /// @param amount Amount to stake
    function stake(uint256 amount) external {
        require(amount > 0, "Amount must be > 0");
        token.safeTransferFrom(msg.sender, address(this), amount);

        StakeInfo storage info = stakes[msg.sender];
        if (!info.active) {
            stakers.push(msg.sender);
            isStaker[msg.sender] = true;
            info.active = true;
        }

        _harvestRewards(msg.sender);
        info.amount += amount;
        info.stakedAt = block.timestamp;
        info.tier = _calculateTier(info.amount);

        totalStaked += amount;

        emit Staked(msg.sender, amount, info.tier);
    }

    /// @notice Unstake tokens
    /// @param amount Amount to unstake
    function unstake(uint256 amount) external {
        StakeInfo storage info = stakes[msg.sender];
        require(info.amount >= amount, "Insufficient stake");

        _harvestRewards(msg.sender);
        info.amount -= amount;
        info.tier = _calculateTier(info.amount);
        totalStaked -= amount;

        token.safeTransfer(msg.sender, amount);

        if (info.amount == 0) {
            info.active = false;
        }

        emit Unstaked(msg.sender, amount);
    }

    /// @notice Claim accumulated rewards
    function claimRewards() external {
        _harvestRewards(msg.sender);
    }

    /// @notice Deposit rewards to the pool (called by FeeDistributor)
    function depositRewards(uint256 amount) external onlyOwner {
        token.safeTransferFrom(msg.sender, address(this), amount);
        rewardPool += amount;
        lastRewardTimestamp = block.timestamp;
    }

    /// @notice Get staker tier
    function getTier(address account) external view returns (Tier) {
        return stakes[account].tier;
    }

    /// @notice Get weighted stake (amount * tier multiplier)
    function getWeightedStake(address account) external view returns (uint256) {
        StakeInfo storage info = stakes[account];
        return info.amount * _tierMultiplier(info.tier);
    }

    /// @notice Get all stakers
    function getStakers() external view returns (address[] memory) {
        return stakers;
    }

    function _harvestRewards(address account) internal {
        StakeInfo storage info = stakes[account];
        if (info.amount == 0 || totalStaked == 0) return;

        uint256 weightedStake = info.amount * _tierMultiplier(info.tier);
        uint256 totalWeighted = totalStaked; // simplified - should be sum of all weighted
        uint256 pending = (rewardPool * weightedStake) / totalWeighted;

        if (pending > 0) {
            info.rewardDebt += pending;
            token.safeTransfer(account, pending);
            rewardPool -= pending;

            emit RewardClaimed(account, pending);
        }
    }

    function _calculateTier(uint256 amount) internal pure returns (Tier) {
        if (amount >= MIN_VALIDATOR) return Tier.Validator;
        if (amount >= MIN_FULL) return Tier.Full;
        if (amount >= MIN_LIGHT) return Tier.Light;
        return Tier.Observer;
    }

    function _tierMultiplier(Tier tier) internal pure returns (uint256) {
        if (tier == Tier.Validator) return TIER_MULTIPLIER_VALIDATOR;
        if (tier == Tier.Full) return TIER_MULTIPLIER_FULL;
        if (tier == Tier.Light) return TIER_MULTIPLIER_LIGHT;
        return 1;
    }
}
