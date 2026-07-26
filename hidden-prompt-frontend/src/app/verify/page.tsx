import { Card, CardContent } from "@/components/ui/card";
import { VerifyGate } from "@/components/verify/VerifyGate";

export default function VerifyPage() {
  return (
    <div className="flex min-h-screen items-center justify-center px-4 py-10">
      <Card className="w-full max-w-md">
        <CardContent className="pt-6">
          <VerifyGate />
        </CardContent>
      </Card>
    </div>
  );
}
