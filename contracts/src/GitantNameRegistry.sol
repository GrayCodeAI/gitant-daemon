// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/access/Ownable.sol";

/// @title GitantNameRegistry
/// @notice ENS-like name registry mapping human-readable names to DIDs
/// @dev Names are registered on Base L2 with annual renewal fees
contract GitantNameRegistry is Ownable {
    struct NameRecord {
        string did;
        address owner;
        uint256 registeredAt;
        uint256 expiresAt;
        bool active;
    }

    mapping(string => NameRecord) public names;
    mapping(address => string[]) public ownerNames;

    uint256 public constant REGISTRATION_FEE = 0.001 ether;
    uint256 public constant RENEWAL_DURATION = 365 days;
    uint256 public constant MIN_NAME_LENGTH = 3;
    uint256 public constant MAX_NAME_LENGTH = 64;

    address public feeRecipient;

    event NameRegistered(string indexed name, string did, address owner);
    event NameRenewed(string indexed name, uint256 expiresAt);
    event NameTransferred(string indexed name, address newOwner);

    constructor(address _feeRecipient) Ownable(msg.sender) {
        feeRecipient = _feeRecipient;
    }

    /// @notice Register a new name
    /// @param name The human-readable name
    /// @param did The DID to associate with the name
    function register(string calldata name, string calldata did) external payable {
        require(bytes(name).length >= MIN_NAME_LENGTH, "Name too short");
        require(bytes(name).length <= MAX_NAME_LENGTH, "Name too long");
        require(!names[name].active || names[name].expiresAt < block.timestamp, "Name taken");
        require(msg.value >= REGISTRATION_FEE, "Insufficient fee");

        names[name] = NameRecord({
            did: did,
            owner: msg.sender,
            registeredAt: block.timestamp,
            expiresAt: block.timestamp + RENEWAL_DURATION,
            active: true
        });

        ownerNames[msg.sender].push(name);

        if (msg.value > REGISTRATION_FEE) {
            (bool success, ) = payable(msg.sender).call{value: msg.value - REGISTRATION_FEE}("");
            require(success, "Refund failed");
        }

        emit NameRegistered(name, did, msg.sender);
    }

    /// @notice Renew a name registration
    /// @param name The name to renew
    function renew(string calldata name) external payable {
        NameRecord storage record = names[name];
        require(record.active, "Name not registered");
        require(record.owner == msg.sender, "Not owner");
        require(msg.value >= REGISTRATION_FEE, "Insufficient fee");

        record.expiresAt = block.timestamp + RENEWAL_DURATION;

        emit NameRenewed(name, record.expiresAt);
    }

    /// @notice Transfer name ownership
    /// @param name The name to transfer
    /// @param newOwner The new owner address
    function transfer(string calldata name, address newOwner) external {
        NameRecord storage record = names[name];
        require(record.active, "Name not registered");
        require(record.owner == msg.sender, "Not owner");
        require(newOwner != address(0), "Invalid address");

        record.owner = newOwner;
        ownerNames[newOwner].push(name);

        emit NameTransferred(name, newOwner);
    }

    /// @notice Resolve a name to its DID
    /// @param name The name to resolve
    /// @return did The associated DID
    function resolve(string calldata name) external view returns (string memory did) {
        NameRecord storage record = names[name];
        require(record.active, "Name not registered");
        require(record.expiresAt >= block.timestamp, "Name expired");
        return record.did;
    }

    /// @notice Check if a name is available
    /// @param name The name to check
    /// @return available Whether the name is available
    function available(string calldata name) external view returns (bool available) {
        return !names[name].active || names[name].expiresAt < block.timestamp;
    }

    /// @notice Get all names for an owner
    /// @param owner The owner address
    /// @return Array of names
    function getNamesByOwner(address owner) external view returns (string[] memory) {
        return ownerNames[owner];
    }

    function setFeeRecipient(address _feeRecipient) external onlyOwner {
        feeRecipient = _feeRecipient;
    }
}
