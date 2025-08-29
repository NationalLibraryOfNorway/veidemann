import {Component, Input} from '@angular/core';
import {ConfigObject} from '../../../../../shared/models/config';
import {MatCheckbox} from '@angular/material/checkbox';
import {DurationFormatPipe} from '../../../../../shared/pipes/duration-format.pipe';
import {FormsModule} from '@angular/forms';

@Component({
  selector: 'app-crawlconfig-preview',
  templateUrl: './crawlconfig-preview.component.html',
  styleUrls: ['./crawlconfig-preview.component.css'],
  imports: [
    DurationFormatPipe,
    FormsModule,
    MatCheckbox
  ],
  standalone: true
})
export class CrawlconfigPreviewComponent {
  @Input()
  configObject: ConfigObject;

  constructor() {
  }
}
