// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import "@openzeppelin/contracts/utils/cryptography/EIP712.sol";

/// @title GitantDIDRegistry
/// @notice On-chain DID document registry for gitant agents
/// @dev Supports did:key, did:web, and did:gitlawb methods
contract GitantDIDRegistry is Ownable, EIP712 {
    using ECDSA for bytes32;

    struct DIDDocument {
        address controller;
        bytes32 publicKey;
        string serviceEndpoint;
        uint256 created;
        uint256 updated;
        bool active;
    }

    mapping(bytes32 => DIDDocument) public dids;
    mapping(address => bytes32[]) public controllerDIDs;

    event DIDRegistered(bytes32 indexed didHash, address controller, bytes32 publicKey);
    event DIDUpdated(bytes32 indexed didHash, address controller);
    event DIDDeactivated(bytes32 indexed didHash);

    constructor() Ownable(msg.sender) EIP712("GitantDIDRegistry", "1") {}

    /// @notice Register a new DID document
    /// @param did The DID string (e.g., "did:gitlawb:z6Mk...")
    /// @param publicKey Ed25519 public key (32 bytes)
    /// @param serviceEndpoint Optional service endpoint URL
    function register(
        string calldata did,
        bytes32 publicKey,
        string calldata serviceEndpoint
    ) external {
        bytes32 didHash = keccak256(abi.encodePacked(did));
        require(!dids[didHash].active, "DID already registered");

        dids[didHash] = DIDDocument({
            controller: msg.sender,
            publicKey: publicKey,
            serviceEndpoint: serviceEndpoint,
            created: block.timestamp,
            updated: block.timestamp,
            active: true
        });

        controllerDIDs[msg.sender].push(didHash);
        emit DIDRegistered(didHash, msg.sender, publicKey);
    }

    /// @notice Update a DID document (controller only)
    /// @param did The DID string
    /// @param newPublicKey New public key
    /// @param newEndpoint New service endpoint
    function update(
        string calldata did,
        bytes32 newPublicKey,
        string calldata newEndpoint
    ) external {
        bytes32 didHash = keccak256(abi.encodePacked(did));
        DIDDocument storage doc = dids[didHash];
        require(doc.active, "DID not active");
        require(doc.controller == msg.sender, "Not controller");

        if (newPublicKey != bytes32(0)) {
            doc.publicKey = newPublicKey;
        }
        if (bytes(newEndpoint).length > 0) {
            doc.serviceEndpoint = newEndpoint;
        }
        doc.updated = block.timestamp;

        emit DIDUpdated(didHash, msg.sender);
    }

    /// @notice Deactivate a DID
    /// @param did The DID string
    function deactivate(string calldata did) external {
        bytes32 didHash = keccak256(abi.encodePacked(did));
        DIDDocument storage doc = dids[didHash];
        require(doc.active, "DID not active");
        require(doc.controller == msg.sender, "Not controller");

        doc.active = false;
        emit DIDDeactivated(didHash);
    }

    /// @notice Resolve a DID to its document
    /// @param did The DID string
    /// @return controller, publicKey, serviceEndpoint, active
    function resolve(string calldata did)
        external
        view
        returns (
            address controller,
            bytes32 publicKey,
            string memory serviceEndpoint,
            bool active
        )
    {
        bytes32 didHash = keccak256(abi.encodePacked(did));
        DIDDocument storage doc = dids[didHash];
        return (doc.controller, doc.publicKey, doc.serviceEndpoint, doc.active);
    }

    /// @notice Get all DIDs for a controller
    /// @param controller The controller address
    /// @return Array of DID hashes
    function getDIDsByController(address controller)
        external
        view
        returns (bytes32[] memory)
    {
        return controllerDIDs[controller];
    }
}
