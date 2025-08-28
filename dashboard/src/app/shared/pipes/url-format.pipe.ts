import {Pipe, PipeTransform} from '@angular/core';
import {DomSanitizer, SafeHtml} from '@angular/platform-browser';
import { SedPipe } from './sed.pipe';

@Pipe({
    name: 'urlFormat',
    standalone: true
})
export class UrlFormatPipe implements PipeTransform {

  constructor(private domSanitizer: DomSanitizer) {
  }

  transform(url: string, stripQueryParams?: boolean): SafeHtml {
    let anchor = ``;
    let urlText = url;

    if (stripQueryParams) {
      urlText = new SedPipe().transform(url);
    }

    anchor = `<a class="formattedUri" href="${url}" target="_blank">${urlText}</a> `;
    return this.domSanitizer.bypassSecurityTrustHtml(anchor);
  }

  private stylize(text: string): string {
    let stylizedText = '';
    if (text && text.length > 0) {
      stylizedText += `<a href="${text}" target="_blank">${text}</a> `;
    }
    return stylizedText;
  }
}
