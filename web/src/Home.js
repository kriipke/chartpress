import React, { useEffect, useState } from 'react';
import Asciidoctor from 'asciidoctor';

function Home() {
  const [htmlContent, setHtmlContent] = useState('');
  const asciidoctor = Asciidoctor();

  useEffect(() => {
    const fetchReadme = async () => {
      const adocUrl = '/chartpress/docs/README.adoc'; // Replace with the relative path to your README.adoc file

      console.log(`[Home] Fetching README.adoc from: ${adocUrl}`);

      try {
        const response = await fetch(adocUrl);
        console.log(`[Home] Fetch response status: ${response.status}`);

        if (!response.ok) {
          const errorMessage = `[Home] Failed to fetch README.adoc. Status: ${response.status} - ${response.statusText}`;
          console.error(errorMessage);
          throw new Error(errorMessage);
        }

        const adocText = await response.text();
        console.log('[Home] Successfully fetched README.adoc content.');

        // Convert AsciiDoc to HTML
        let html;
        try {
          html = asciidoctor.convert(adocText);
          console.log('[Home] Successfully converted README.adoc to HTML.');
        } catch (conversionError) {
          const errorMessage = '[Home] Error converting AsciiDoc to HTML.';
          console.error(errorMessage, conversionError);
          throw conversionError;
        }

        setHtmlContent(html);
      } catch (error) {
        console.error('[Home] Error occurred:', error);
        setHtmlContent('<p>Failed to load the README content. Please try again later.</p>');
      }
    };

    fetchReadme();
  }, []);

  return (
    <div className="home-container">
      <h2>Welcome to ChartPress</h2>
      {/* Render the converted HTML content */}
      <div dangerouslySetInnerHTML={{ __html: htmlContent }} />
    </div>
  );
}

export default Home;
