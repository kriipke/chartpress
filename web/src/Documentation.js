import React, { useEffect, useState } from 'react';
import Asciidoctor from 'asciidoctor';

function Documentation() {
  const [htmlContent, setHtmlContent] = useState('');
  const asciidoctor = Asciidoctor();

  useEffect(() => {
    const fetchDocs = async () => {
      const adocUrl = '/chartpress/docs/README.adoc'; // Replace with your desired .adoc file URL

      console.log(`[Documentation] Starting fetch for: ${adocUrl}`);

      try {
        // Start fetching the .adoc file
        const response = await fetch(adocUrl);
        console.log(`[Documentation] Fetch response status: ${response.status}`);

        // Check if the response status is not OK
        if (!response.ok) {
          const errorMessage = `[Documentation] Failed to fetch documentation. Status: ${response.status} - ${response.statusText}`;
          console.error(errorMessage);
          throw new Error(errorMessage);
        }

        // Parse the AsciiDoc content
        const adocText = await response.text();
        console.log(`[Documentation] Successfully fetched ${adocUrl} content.`);

        // Convert AsciiDoc to HTML
        let html;
        try {
          html = asciidoctor.convert(adocText);
          console.log('[Documentation] Successfully converted ${adocUrl}.adoc to HTML.');
        } catch (conversionError) {
          const errorMessage = '[Documentation] Error converting AsciiDoc to HTML.';
          console.error(errorMessage, conversionError);
          throw conversionError;
        }

        // Update the state with the converted HTML content
        setHtmlContent(html);
      } catch (error) {
        // Log the error and set a fallback message
        console.error('[Documentation] Error occurred:', error);
        setHtmlContent('<p>Failed to load documentation. Please try again later.</p>');
      }
    };

    // Execute the fetchDocs function
    fetchDocs();
  }, []);

  return (
    <div className="documentation-container">
      <h2>Documentation</h2>
      {/* Render the converted HTML content */}
      <div dangerouslySetInnerHTML={{ __html: htmlContent }} />
    </div>
  );
}

export default Documentation;
