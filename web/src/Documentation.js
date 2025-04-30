import React, { useEffect, useState } from 'react';
import Asciidoctor from 'asciidoctor';

function Documentation() {
  const [htmlContent, setHtmlContent] = useState('');
  const asciidoctor = Asciidoctor();

  useEffect(() => {
    // Fetch the .adoc file from the remote URL
    const fetchDocs = async () => {
      const adocUrl = '/chartpress/docs/README.adoc'; // Replace with your desired .adoc file URL

      try {
        const response = await fetch(adocUrl);
        if (!response.ok) {
          throw new Error('Failed to fetch documentation');
        }

        const adocText = await response.text();

        // Convert the AsciiDoc content to HTML
        const html = asciidoctor.convert(adocText);
        setHtmlContent(html);
      } catch (error) {
        console.error('Error fetching documentation:', error);
        setHtmlContent('<p>Failed to load documentation. Please try again later.</p>');
      }
    };

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
