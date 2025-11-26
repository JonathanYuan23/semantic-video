import { useEffect, useState, useRef } from "react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { CloudStatus } from "@/components/CloudStatus";
import { SearchBar } from "@/components/SearchBar";
import { VideoResult } from "@/components/VideoResult";
import { EmptyState } from "@/components/EmptyState";
import { VideoLibrary } from "@/components/VideoLibrary";
import { IndexingProgress } from "@/components/IndexingProgress";
import { VideoPlayer } from "@/components/VideoPlayer";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";
import { Loader2, Library, Search as SearchIcon, Activity } from "lucide-react";
import { API_BASE, searchVideos, SearchMatch } from "@/lib/api";

const Index = () => {
  const [results, setResults] = useState<SearchMatch[]>([]);
  const [loading, setLoading] = useState(false);
  const [hasSearched, setHasSearched] = useState(false);
  const [lastQuery, setLastQuery] = useState("");
  const [selectedVideo, setSelectedVideo] = useState<{ videoId?: string; filename: string; timestamp: number } | null>(null);
  const [videoUrl, setVideoUrl] = useState<string>("");
  const playerRef = useRef<HTMLDivElement>(null);

  const handleSearch = async (query: string) => {
    setLoading(true);
    setLastQuery(query);
    setHasSearched(true);

    try {
      const matches = await searchVideos(query, 20, 0);
      setResults(matches);
      if (matches.length === 0) {
        toast.info("No matching videos found");
      } else {
        toast.success(`Found ${matches.length} match(es)`);
      }
    } catch (error) {
      console.error("Search error:", error);
      toast.error("Could not connect to search service");
      setResults([]);
    } finally {
      setLoading(false);
    }
  };

  const handleOpenTimestamp = (filename: string, timestamp: number, videoId?: string) => {
    setSelectedVideo({ filename, timestamp, videoId });
    toast.success(`Opening ${filename} at ${Math.round(timestamp)}s`);
    requestAnimationFrame(() => {
      playerRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
    });
  };

  const handleSelectVideo = (filename: string, videoId?: string) => {
    setSelectedVideo({ filename, timestamp: 0, videoId });
    toast.success(`Loading ${filename.split("/").pop()}`);
  };

  useEffect(() => {
    if (!selectedVideo) {
      setVideoUrl("");
      return;
    }

    if (selectedVideo.videoId) {
      setVideoUrl(`${API_BASE}/videos/${selectedVideo.videoId}/file`);
      return;
    }

    const name = selectedVideo.filename || "";
    if (name.startsWith("http")) {
      setVideoUrl(name);
    } else if (name.startsWith("/")) {
      setVideoUrl(`file://${name}`);
    } else {
      setVideoUrl(name);
    }
  }, [selectedVideo]);

  return (
    <div className="min-h-screen bg-background">
      <div className="container mx-auto px-6 py-8 max-w-7xl">
        {/* Header */}
        <header className="mb-8">
            <div className="flex items-center justify-between mb-6">
              <div>
                <h1 className="text-4xl font-bold mb-2 bg-gradient-to-r from-primary to-primary/60 bg-clip-text text-transparent">
                  Semantic Video Search
                </h1>
              <p className="text-muted-foreground">
                Index and search through your videos using AI
              </p>
            </div>
          </div>

          <CloudStatus />
        </header>

        {/* Main Content */}
        <Tabs defaultValue="search" className="space-y-6">
          <TabsList className="grid w-full grid-cols-3 max-w-md">
            <TabsTrigger value="search" className="flex items-center gap-2">
              <SearchIcon className="h-4 w-4" />
              Search
            </TabsTrigger>
            <TabsTrigger value="library" className="flex items-center gap-2">
              <Library className="h-4 w-4" />
              Library
            </TabsTrigger>
            <TabsTrigger value="indexing" className="flex items-center gap-2">
              <Activity className="h-4 w-4" />
              Indexing
            </TabsTrigger>
          </TabsList>

          {/* Search Tab */}
          <TabsContent value="search" className="space-y-6">
            <SearchBar onSearch={handleSearch} loading={loading} />

            {selectedVideo && (
              <div className="space-y-3" ref={playerRef}>
                <div className="flex items-center justify-between">
                  <h3 className="text-lg font-semibold">Now Playing</h3>
                  <Badge variant="secondary" className="font-mono">
                    {selectedVideo.timestamp}s
                  </Badge>
                </div>
                <VideoPlayer 
                  videoUrl={videoUrl}
                  startTime={selectedVideo.timestamp}
                  filename={selectedVideo.filename}
                  onClose={() => setSelectedVideo(null)}
                />
              </div>
            )}

            {loading && (
              <Card className="p-12 gradient-card border-border/50">
                <div className="text-center">
                  <Loader2 className="h-12 w-12 animate-spin text-primary mx-auto mb-4" />
                  <p className="text-muted-foreground">Searching through your videos...</p>
                </div>
              </Card>
            )}

            {!loading && !hasSearched && <EmptyState type="initial" />}

            {!loading && hasSearched && results.length === 0 && (
              <EmptyState type="no-results" query={lastQuery} />
            )}

            {!loading && results.length > 0 && (
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <h2 className="text-xl font-semibold">Search Results</h2>
          <Badge variant="secondary">
            {results.length} match{results.length !== 1 ? 'es' : ''}
          </Badge>
        </div>

        <div className="space-y-3">
          {results.map((result, idx) => (
            <VideoResult
              key={idx}
              videoId={result.videoId}
              filename={result.videoPath || result.videoId}
              timestamps={[result.timestamp]}
              relevanceScore={result.relevanceScore}
              rank={idx + 1}
              bestScore={results[0]?.relevanceScore}
              onOpenTimestamp={(file, ts, vid) => handleOpenTimestamp(file, ts, vid)}
            />
          ))}
        </div>
      </div>
    )}
          </TabsContent>

          {/* Library Tab */}
          <TabsContent value="library">
            <VideoLibrary 
              onSelectVideo={(video) => handleSelectVideo(video.path, video.id)}
            />
          </TabsContent>

          {/* Indexing Tab */}
          <TabsContent value="indexing">
            <IndexingProgress />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
};

export default Index;
