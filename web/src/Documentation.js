import React, { useEffect, useState } from 'react';
import Asciidoctor from 'asciidoctor';

function Documentation() {
  const [htmlContent, setHtmlContent] = useState('');
  const asciidoctor = Asciidoctor();

  useEffect(() => {
    // Fetch all .adoc files from the /docs directory
    const fetchDocs = async () => {
      try {
        const response = await fetch('/chartpress/docs'); // Adjust the path if needed
        if (!response.ok) {
          throw new Error('Failed to fetch documentation');
        }
        const adocText = await response.text();

        // Convert AsciiDoc to HTML
        const html = asciidoctor.convert(adocText);
        setHtmlContent(html);
      } catch (error) {
        console.error('Error fetching documentation:', error);
        setHtmlContent('<p>Failed to load documentation.</p>');
      }
    };

    fetchDocs();
  }, []);

  return (
    <div className="documentation-container">
      <h2>Documentation</h2>
      {/* Render compiled HTML content */}
      <div dangerouslySetInnerHTML={{ __html: htmlContent }} />
    </div>
  );
}

export default Documentation;
