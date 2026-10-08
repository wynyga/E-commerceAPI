import { Suspense } from "react";

import Catalogue from "@/components/Catalogue";

export default function Home() {
  return (
    <Suspense>
      <Catalogue />
    </Suspense>
  );
}
