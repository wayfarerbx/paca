import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// Build a minimal mock socket that tracks calls.
function createMockSocket() {
	return {
		connected: false,
		connect: vi.fn(),
		disconnect: vi.fn(),
		emit: vi.fn(),
		on: vi.fn(),
		off: vi.fn(),
	};
}

const { mockSocketFactory, mockSocket } = vi.hoisted(() => {
	const socket = createMockSocket();
	return {
		mockSocketFactory: vi.fn(() => socket),
		mockSocket: socket,
	};
});

vi.mock("socket.io-client", () => ({
	io: mockSocketFactory,
}));

// Import after mocking so the module-level `io` call uses our mock.
import {
	connectSocket,
	disconnectSocket,
	getSocket,
	joinProject,
	leaveProject,
} from "./socket-client";

describe("socket-client", () => {
	beforeEach(() => {
		// Reset mock state before each test.
		vi.clearAllMocks();
		mockSocket.connected = false;
		// Ensure internal singleton is cleared between tests.
		disconnectSocket();
	});

	afterEach(() => {
		disconnectSocket();
	});

	describe("connectSocket", () => {
		it("creates a new socket on first call", () => {
			const socket = connectSocket();

			expect(mockSocketFactory).toHaveBeenCalledTimes(1);
			expect(socket).toBe(mockSocket);
		});

		it("reuses the existing socket when already connected", () => {
			mockSocket.connected = true;

			connectSocket();
			const second = connectSocket();

			expect(mockSocketFactory).toHaveBeenCalledTimes(1);
			expect(second).toBe(mockSocket);
		});

		it("calls connect() on an existing disconnected socket instead of creating a new one", () => {
			// First call creates the socket.
			connectSocket();
			expect(mockSocketFactory).toHaveBeenCalledTimes(1);

			// Simulate disconnected state (connected = false, socket exists).
			mockSocket.connected = false;

			// Second call should reuse and call connect().
			connectSocket();
			expect(mockSocketFactory).toHaveBeenCalledTimes(1);
			expect(mockSocket.connect).toHaveBeenCalledTimes(1);
		});
	});

	describe("disconnectSocket", () => {
		it("disconnects and nullifies the socket", () => {
			connectSocket();
			disconnectSocket();

			expect(mockSocket.disconnect).toHaveBeenCalledTimes(1);
			expect(getSocket()).toBeNull();
		});

		it("is a no-op when no socket exists", () => {
			// Should not throw.
			expect(() => disconnectSocket()).not.toThrow();
		});
	});

	describe("getSocket", () => {
		it("returns null when no socket is created", () => {
			expect(getSocket()).toBeNull();
		});

		it("returns the current socket after connecting", () => {
			connectSocket();
			expect(getSocket()).toBe(mockSocket);
		});
	});

	describe("joinProject", () => {
		it("emits a join event with the projectId", () => {
			connectSocket();
			joinProject("proj-123");

			expect(mockSocket.emit).toHaveBeenCalledWith("join", {
				projectId: "proj-123",
			});
		});

		it("is a no-op when no socket is connected", () => {
			// No connectSocket() call — socket is null.
			joinProject("proj-123");
			expect(mockSocket.emit).not.toHaveBeenCalled();
		});
	});

	describe("leaveProject", () => {
		it("emits a leave event with the projectId", () => {
			connectSocket();
			leaveProject("proj-123");

			expect(mockSocket.emit).toHaveBeenCalledWith("leave", {
				projectId: "proj-123",
			});
		});

		it("is a no-op when no socket is connected", () => {
			leaveProject("proj-123");
			expect(mockSocket.emit).not.toHaveBeenCalled();
		});
	});
});
