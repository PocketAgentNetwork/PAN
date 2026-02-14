// -----------------------------------------------------------
// 🦉 Config
// -----------------------------------------------------------

export const Config = {
    PORT: parseInt(process.env.PORT || "8080", 10),
    SECRET_KEY: process.env.A2A_KEY || "owl-secret-2026",

    // Limits
    RATE_LIMIT_WINDOW_MS: 1000,
    MAX_MSGS_PER_WINDOW: 5,
    MAX_IDLE_TIME_MS: 30000, // Not enforced yet

    // Formatting
    LOG_COLORS: true
};
