import {Component, Input} from '@angular/core';
import {ConfigObject} from '../../../../../shared/models/config';
import {DurationFormatPipe} from '../../../../../shared/pipes/duration-format.pipe';
import {MatLabel} from '@angular/material/form-field';

@Component({
  selector: 'app-crawlhostgroupconfig-preview',
  templateUrl: './crawlhostgroupconfig-preview.component.html',
  styleUrls: ['./crawlhostgroupconfig-preview.component.css'],
  imports: [
    DurationFormatPipe,
    MatLabel
  ],
  standalone: true
})
export class CrawlhostgroupconfigPreviewComponent {
  @Input()
  configObject: ConfigObject;

  constructor() {
  }

}
