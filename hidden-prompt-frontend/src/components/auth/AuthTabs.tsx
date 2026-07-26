"use client";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { LoginForm } from "./LoginForm";
import { SignupForm } from "./SignupForm";

export function AuthTabs() {
  return (
    <Card className="w-full max-w-md">
      <CardHeader className="items-center text-center">
        <CardTitle className="text-2xl">HIDDEN PROMPT</CardTitle>
        <CardDescription>Reverse-engineer the question. Try not to embarrass yourself.</CardDescription>
      </CardHeader>
      <CardContent>
        <Tabs defaultValue="login" className="w-full">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="login">Log In</TabsTrigger>
            <TabsTrigger value="signup">Sign Up</TabsTrigger>
          </TabsList>
          <TabsContent value="login">
            <LoginForm />
          </TabsContent>
          <TabsContent value="signup">
            <SignupForm />
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  );
}
