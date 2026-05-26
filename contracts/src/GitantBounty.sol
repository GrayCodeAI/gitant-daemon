// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";

/// @title GitantBounty
/// @notice On-chain bounty escrow with 5% protocol fee
/// @dev Agents can create bounties, claim work, and receive trustless payments
contract GitantBounty is Ownable {
    using SafeERC20 for IERC20;

    enum BountyStatus {
        Open,
        Claimed,
        Submitted,
        Approved,
        Cancelled,
        Disputed
    }

    struct Bounty {
        address creator;
        address claimant;
        IERC20 token;
        uint256 amount;
        uint256 protocolFee;
        string repo;
        string issueId;
        string description;
        BountyStatus status;
        uint256 createdAt;
        uint256 claimedAt;
        uint256 deadline;
    }

    uint256 public constant PROTOCOL_FEE_BPS = 500; // 5%
    uint256 public constant BPS_DENOMINATOR = 10000;

    uint256 public nextBountyId;
    mapping(uint256 => Bounty) public bounties;
    mapping(string => uint256[]) public repoBounties;
    mapping(address => uint256[]) public creatorBounties;
    mapping(address => uint256[]) public claimantBounties;

    address public feeRecipient;

    event BountyCreated(uint256 indexed id, address creator, string repo, uint256 amount);
    event BountyClaimed(uint256 indexed id, address claimant);
    event BountySubmitted(uint256 indexed id, address claimant);
    event BountyApproved(uint256 indexed id, uint256 payout);
    event BountyCancelled(uint256 indexed id);
    event BountyDisputed(uint256 indexed id);

    constructor(address _feeRecipient) Ownable(msg.sender) {
        feeRecipient = _feeRecipient;
    }

    /// @notice Create a bounty with ERC-20 token escrow
    /// @param token The ERC-20 token address
    /// @param amount The bounty amount
    /// @param repo The repository name
    /// @param issueId The issue ID
    /// @param description Bounty description
    function create(
        address token,
        uint256 amount,
        string calldata repo,
        string calldata issueId,
        string calldata description,
        uint256 deadline
    ) external returns (uint256) {
        require(amount > 0, "Amount must be > 0");
        require(deadline > block.timestamp, "Deadline must be future");

        uint256 fee = (amount * PROTOCOL_FEE_BPS) / BPS_DENOMINATOR;
        uint256 escrowAmount = amount;

        IERC20(token).safeTransferFrom(msg.sender, address(this), escrowAmount);

        uint256 bountyId = nextBountyId++;
        bounties[bountyId] = Bounty({
            creator: msg.sender,
            claimant: address(0),
            token: IERC20(token),
            amount: amount,
            protocolFee: fee,
            repo: repo,
            issueId: issueId,
            description: description,
            status: BountyStatus.Open,
            createdAt: block.timestamp,
            claimedAt: 0,
            deadline: deadline
        });

        repoBounties[repo].push(bountyId);
        creatorBounties[msg.sender].push(bountyId);

        emit BountyCreated(bountyId, msg.sender, repo, amount);
        return bountyId;
    }

    /// @notice Claim a bounty (agent commits to completing it)
    /// @param bountyId The bounty ID
    function claim(uint256 bountyId) external {
        Bounty storage bounty = bounties[bountyId];
        require(bounty.status == BountyStatus.Open, "Not open");
        require(bounty.deadline > block.timestamp, "Expired");
        require(msg.sender != bounty.creator, "Creator cannot claim");

        bounty.claimant = msg.sender;
        bounty.status = BountyStatus.Claimed;
        bounty.claimedAt = block.timestamp;

        claimantBounties[msg.sender].push(bountyId);

        emit BountyClaimed(bountyId, msg.sender);
    }

    /// @notice Submit work for a claimed bounty
    /// @param bountyId The bounty ID
    function submit(uint256 bountyId) external {
        Bounty storage bounty = bounties[bountyId];
        require(bounty.status == BountyStatus.Claimed, "Not claimed");
        require(msg.sender == bounty.claimant, "Not claimant");

        bounty.status = BountyStatus.Submitted;

        emit BountySubmitted(bountyId, msg.sender);
    }

    /// @notice Approve submission and release payment
    /// @param bountyId The bounty ID
    function approve(uint256 bountyId) external {
        Bounty storage bounty = bounties[bountyId];
        require(bounty.status == BountyStatus.Submitted, "Not submitted");
        require(msg.sender == bounty.creator, "Not creator");

        uint256 payout = bounty.amount - bounty.protocolFee;

        bounty.token.safeTransfer(feeRecipient, bounty.protocolFee);
        bounty.token.safeTransfer(bounty.claimant, payout);

        bounty.status = BountyStatus.Approved;

        emit BountyApproved(bountyId, payout);
    }

    /// @notice Cancel a bounty and refund escrow
    /// @param bountyId The bounty ID
    function cancel(uint256 bountyId) external {
        Bounty storage bounty = bounties[bountyId];
        require(bounty.status == BountyStatus.Open || bounty.status == BountyStatus.Claimed, "Cannot cancel");
        require(msg.sender == bounty.creator, "Not creator");

        if (bounty.status == BountyStatus.Claimed) {
            require(block.timestamp > bounty.deadline, "Deadline not passed");
        }

        bounty.token.safeTransfer(bounty.creator, bounty.amount);
        bounty.status = BountyStatus.Cancelled;

        emit BountyCancelled(bountyId);
    }

    /// @notice Dispute a submission (triggers arbitration)
    /// @param bountyId The bounty ID
    function dispute(uint256 bountyId) external {
        Bounty storage bounty = bounties[bountyId];
        require(
            bounty.status == BountyStatus.Submitted || bounty.status == BountyStatus.Claimed,
            "Cannot dispute"
        );
        require(msg.sender == bounty.creator || msg.sender == bounty.claimant, "Not party");

        bounty.status = BountyStatus.Disputed;
        emit BountyDisputed(bountyId);
    }

    /// @notice Get bounties for a repository
    function getRepoBounties(string calldata repo)
        external
        view
        returns (uint256[] memory)
    {
        return repoBounties[repo];
    }

    /// @notice Get bounties created by an address
    function getCreatorBounties(address creator)
        external
        view
        returns (uint256[] memory)
    {
        return creatorBounties[creator];
    }

    function setFeeRecipient(address _feeRecipient) external onlyOwner {
        feeRecipient = _feeRecipient;
    }
}
