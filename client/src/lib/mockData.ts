// Mock video data for testing
export const mockVideos = [
  {
    id: "vid-001",
    filename: "nature_documentary_2024.mp4",
    path: "/Users/demo/Videos/nature_documentary_2024.mp4",
    duration: 3600,
    size: 2147483648,
    status: "indexed",
    indexedAt: "2024-03-15T10:30:00Z",
  },
  {
    id: "vid-002",
    filename: "cooking_tutorial_pasta.mp4",
    path: "/Users/demo/Videos/cooking_tutorial_pasta.mp4",
    duration: 1200,
    size: 524288000,
    status: "indexed",
    indexedAt: "2024-03-14T15:20:00Z",
  },
  {
    id: "vid-003",
    filename: "travel_vlog_japan.mp4",
    path: "/Users/demo/Videos/travel_vlog_japan.mp4",
    duration: 2400,
    size: 1073741824,
    status: "pending",
    indexedAt: null,
  },
  {
    id: "vid-004",
    filename: "coding_workshop_react.mp4",
    path: "/Users/demo/Videos/coding_workshop_react.mp4",
    duration: 5400,
    size: 3221225472,
    status: "indexed",
    indexedAt: "2024-03-13T09:15:00Z",
  },
];

export const mockSearchResults = {
  sunset: [
    {
      filename: "nature_documentary_2024.mp4",
      timestamps: [245, 612, 1023, 1456, 2890, 3201],
      relevanceScore: 0.92,
    },
    {
      filename: "travel_vlog_japan.mp4",
      timestamps: [1850, 2156],
      relevanceScore: 0.78,
    },
  ],
  cooking: [
    {
      filename: "cooking_tutorial_pasta.mp4",
      timestamps: [45, 230, 456, 789, 1050],
      relevanceScore: 0.95,
    },
  ],
  mountains: [
    {
      filename: "nature_documentary_2024.mp4",
      timestamps: [890, 1345, 2234, 3045],
      relevanceScore: 0.88,
    },
    {
      filename: "travel_vlog_japan.mp4",
      timestamps: [345, 890, 1234],
      relevanceScore: 0.82,
    },
  ],
  code: [
    {
      filename: "coding_workshop_react.mp4",
      timestamps: [120, 450, 890, 1456, 2345, 3678, 4567, 5123],
      relevanceScore: 0.93,
    },
  ],
};

export const mockJobs = [
  {
    id: "job-001",
    videoId: "vid-003",
    filename: "travel_vlog_japan.mp4",
    status: "processing",
    progress: 67,
    startedAt: "2024-03-16T14:30:00Z",
    framesExtracted: 2400,
    totalFrames: 3600,
  },
];

// Mock video URLs (using a placeholder video service)
