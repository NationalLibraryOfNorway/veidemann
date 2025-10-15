import {Component} from '@angular/core';
import {environment} from '../../../../environments/environment';
import {MatDivider, MatList, MatListSubheaderCssMatStyler} from '@angular/material/list';
import {MatDialogModule} from '@angular/material/dialog';
import {MatButtonModule} from '@angular/material/button';

@Component({
  selector: 'app-about-dialog',
  templateUrl: './about-dialog.component.html',
  styleUrls: ['./about-dialog.component.css'],
  imports: [
    MatList,
    MatDivider,
    MatDialogModule,
    MatButtonModule,
    MatListSubheaderCssMatStyler
  ],
  standalone: true
})

export class AboutDialogComponent {
  readonly environment = environment;

  constructor() {
  }
}
