FEN stands for Forsyth-Edwards Notation. It is a single line of text that describes the exact state of a chessboard. It allows chess computers and players to save, share, and load specific board positions instantly without needing a full game history.A standard FEN string contains exactly six pieces of information, separated by spaces:

- Piece Placement: The layout of the pieces on the board, starting from the 8th rank (top) down to the 1st rank (bottom). Slashes separate each row. White pieces are uppercase, black are lowercase, and numbers represent consecutive empty squares.
- Active Color: Who has the next turn (w for White, b for Black).
- Castling Rights: Which sides can still castle. (K and Q for White kingside/queenside, k and q for Black). A - means neither side can castle.
- En Passant Square: The square a pawn could capture en passant. If no en passant is possible, this is marked as -.
- Halfmove Clock: The number of moves since the last pawn advance or piece capture. This is used to track the 50-move draw rule.
- Fullmove Number: The total number of moves in the game. It increases by 1 after Black makes their turn.

**Example:** The starting position of a chess game reads as follows in FEN:
rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1
