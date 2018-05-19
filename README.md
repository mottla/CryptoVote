# CryptoVote
CryptoVote is a protocol that allows
the creation and execution of fully decentralized election processes by combining the blockchain with elaborated cryptography.

*It aims to provide:* 

1. Authenticity and forgery resistance of votes
2. Guaranteed anonymity of the voter s.t. no one can tell, if one participated in a
voting or not.
3. Non-Excludability of any authorized voter.
4. No duplication and multiple voting.
5. Hiding the state of a voting until the voting period is over.

It uses crypto from Crypto Note 2.0 (Nicolas van Saberhagen) (ring signature and stealth address).
Operations on the ed25519 curve taken from https://github.com/decred/dcrd

Currently it should be used for educational purposes only!

*Missing*
1. No underlying database
2. No automated neighbor finding
3. No DOS protection
4. No elegant handling of orphan blocks
5. No merkle tree. One block holds at most one transaction.

