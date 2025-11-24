import { useEffect, useState } from "react";
import { Cloud, CloudOff, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";
import { authenticateCloud, getCloudStatus, CloudStatus as CloudStatusType } from "@/lib/api";

export const CloudStatus = () => {
  const [status, setStatus] = useState<CloudStatusType | null>(null);
  const [loading, setLoading] = useState(true);
  const [showAuth, setShowAuth] = useState(false);
  const [token, setToken] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const fetchStatus = async () => {
    try {
      const data = await getCloudStatus();
      setStatus(data);
    } catch (error) {
      console.error("Failed to fetch cloud status:", error);
      toast.error("Unable to load cloud status");
    } finally {
      setLoading(false);
    }
  };

  const handleAuth = async () => {
    try {
      setSubmitting(true);
      await authenticateCloud(token);
      toast.success("Cloud connected successfully");
      setShowAuth(false);
      setToken("");
      fetchStatus();
    } catch (error) {
      toast.error("Failed to connect to cloud");
    } finally {
      setSubmitting(false);
    }
  };

  useEffect(() => {
    fetchStatus();
    const interval = setInterval(fetchStatus, 10000);
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <Card className="p-4 gradient-card">
        <div className="flex items-center gap-3">
          <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          <span className="text-sm text-muted-foreground">Checking cloud status...</span>
        </div>
      </Card>
    );
  }

  return (
    <Card className="p-4 gradient-card border-border/50">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          {status?.connected ? (
            <>
              <Cloud className="h-5 w-5 text-primary" />
              <div>
                <p className="text-sm font-medium">Cloud Connected</p>
                {status.pendingBatches > 0 && (
                  <p className="text-xs text-muted-foreground">
                    {status.pendingBatches} pending uploads
                  </p>
                )}
                {status.lastSuccessfulUpload && (
                  <p className="text-xs text-muted-foreground">
                    Last sync: {new Date(status.lastSuccessfulUpload).toLocaleString()}
                  </p>
                )}
              </div>
            </>
          ) : (
            <>
              <CloudOff className="h-5 w-5 text-muted-foreground" />
              <div>
                <p className="text-sm font-medium text-muted-foreground">Not Connected</p>
                <p className="text-xs text-muted-foreground">Cloud authentication required</p>
              </div>
            </>
          )}
        </div>
        
        {!status?.connected && (
          <Button 
            variant="outline" 
            size="sm"
            onClick={() => setShowAuth(!showAuth)}
            className="border-primary/30 text-primary hover:bg-primary/10"
          >
            Connect
          </Button>
        )}
      </div>

      {showAuth && (
        <div className="mt-4 pt-4 border-t border-border/50 space-y-3">
          <div className="space-y-2">
            <Label htmlFor="token" className="text-xs">Access Token</Label>
            <Input
              id="token"
              type="password"
              placeholder="Enter your access token"
              value={token}
              onChange={(e) => setToken(e.target.value)}
              className="bg-background/50"
            />
          </div>
          <div className="flex gap-2">
            <Button 
              onClick={handleAuth} 
              size="sm"
              className="gradient-primary text-primary-foreground flex-1"
            >
              Authenticate
            </Button>
            <Button 
              variant="ghost" 
              size="sm"
              onClick={() => setShowAuth(false)}
            >
              Cancel
            </Button>
          </div>
        </div>
      )}
    </Card>
  );
};
